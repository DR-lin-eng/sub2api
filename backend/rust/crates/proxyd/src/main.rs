use std::convert::Infallible;
use std::fs;
use std::io;
use std::os::unix::fs::FileTypeExt;
use std::path::Path;
use std::sync::atomic::{AtomicI64, Ordering};

use bytes::Bytes;
use fastwebsockets::{after_handshake_split, OpCode, Role, WebSocketError};
use http_body_util::combinators::BoxBody;
use http_body_util::{BodyExt, Full};
use hyper::body::Incoming;
use hyper::client::conn::http1;
use hyper::header::{CONTENT_TYPE, HeaderValue};
use hyper::service::service_fn;
use hyper::upgrade;
use hyper::{Method, Request, Response, StatusCode};
use hyper_util::rt::{TokioExecutor, TokioIo};
use hyper_util::server::conn::auto::Builder as AutoBuilder;
use serde::Serialize;
use tokio::net::UnixListener;
use tokio::time::{timeout, Duration};

const DEFAULT_SOCKET_PATH: &str = "/tmp/sub2api-rust-sidecar.sock";
const DEFAULT_UPSTREAM_SOCKET_PATH: &str = "/tmp/sub2api-rust-upstream.sock";
const BYPASS_HEADER: &str = "x-sub2api-rust-sidecar-bypass";
type ProxyBody = BoxBody<Bytes, hyper::Error>;

#[derive(Serialize)]
struct HealthPayload<'a> {
    status: &'a str,
    service: &'a str,
    version: &'a str,
    active_connections: i64,
    total_connections: i64,
    active_upgrades: i64,
    total_upgrades: i64,
    total_requests: i64,
    total_request_errors: i64,
    upstream_unavailable_total: i64,
    upstream_handshake_failed_total: i64,
    upstream_request_failed_total: i64,
    upgrade_errors_total: i64,
    relay_bytes_downstream_to_upstream: i64,
    relay_bytes_upstream_to_downstream: i64,
    relay_frames_downstream_to_upstream: i64,
    relay_frames_upstream_to_downstream: i64,
    relay_close_frames_total: i64,
    relay_ping_frames_total: i64,
    relay_pong_frames_total: i64,
}

static ACTIVE_CONNECTIONS: AtomicI64 = AtomicI64::new(0);
static TOTAL_CONNECTIONS: AtomicI64 = AtomicI64::new(0);
static ACTIVE_UPGRADES: AtomicI64 = AtomicI64::new(0);
static TOTAL_UPGRADES: AtomicI64 = AtomicI64::new(0);
static TOTAL_REQUESTS: AtomicI64 = AtomicI64::new(0);
static TOTAL_REQUEST_ERRORS: AtomicI64 = AtomicI64::new(0);
static UPSTREAM_UNAVAILABLE_TOTAL: AtomicI64 = AtomicI64::new(0);
static UPSTREAM_HANDSHAKE_FAILED_TOTAL: AtomicI64 = AtomicI64::new(0);
static UPSTREAM_REQUEST_FAILED_TOTAL: AtomicI64 = AtomicI64::new(0);
static UPGRADE_ERRORS_TOTAL: AtomicI64 = AtomicI64::new(0);
static RELAY_BYTES_DOWNSTREAM_TO_UPSTREAM: AtomicI64 = AtomicI64::new(0);
static RELAY_BYTES_UPSTREAM_TO_DOWNSTREAM: AtomicI64 = AtomicI64::new(0);
static RELAY_FRAMES_DOWNSTREAM_TO_UPSTREAM: AtomicI64 = AtomicI64::new(0);
static RELAY_FRAMES_UPSTREAM_TO_DOWNSTREAM: AtomicI64 = AtomicI64::new(0);
static RELAY_CLOSE_FRAMES_TOTAL: AtomicI64 = AtomicI64::new(0);
static RELAY_PING_FRAMES_TOTAL: AtomicI64 = AtomicI64::new(0);
static RELAY_PONG_FRAMES_TOTAL: AtomicI64 = AtomicI64::new(0);

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let socket_path =
        std::env::var("SUB2API_RUST_SIDECAR_SOCKET").unwrap_or_else(|_| DEFAULT_SOCKET_PATH.to_string());

    cleanup_socket_path(&socket_path)?;

    let listener = UnixListener::bind(&socket_path)?;
    eprintln!("proxyd listening on {}", socket_path);

    loop {
        tokio::select! {
            accept = listener.accept() => {
                let (stream, _) = accept?;
                tokio::spawn(async move {
                    ACTIVE_CONNECTIONS.fetch_add(1, Ordering::Relaxed);
                    TOTAL_CONNECTIONS.fetch_add(1, Ordering::Relaxed);
                    let io = TokioIo::new(stream);
                    let svc = service_fn(handle_request);
                    let builder = AutoBuilder::new(TokioExecutor::new());
                    if let Err(err) = builder.serve_connection_with_upgrades(io, svc).await {
                        eprintln!("proxyd connection error: {}", err);
                    }
                    ACTIVE_CONNECTIONS.fetch_sub(1, Ordering::Relaxed);
                });
            }
            signal = tokio::signal::ctrl_c() => {
                if let Err(err) = signal {
                    eprintln!("proxyd shutdown signal error: {}", err);
                } else {
                    eprintln!("proxyd shutdown requested");
                }
                break;
            }
        }
    }

    drop(listener);
    cleanup_socket_path(&socket_path)?;
    Ok(())
}

async fn handle_request(req: Request<Incoming>) -> Result<Response<ProxyBody>, Infallible> {
    let upstream_socket = std::env::var("SUB2API_RUST_UPSTREAM_SOCKET")
        .unwrap_or_else(|_| DEFAULT_UPSTREAM_SOCKET_PATH.to_string());
    Ok(handle_request_with_upstream(req, upstream_socket).await)
}

async fn handle_request_with_upstream(
    req: Request<Incoming>,
    upstream_socket: String,
) -> Response<ProxyBody> {
    let response = match (req.method(), req.uri().path()) {
        (&Method::GET, "/health") | (&Method::GET, "/healthz") => json_response(StatusCode::OK, &current_health_payload()),
        _ => proxy_request(req, &upstream_socket).await,
    };
    response
}

fn current_health_payload() -> HealthPayload<'static> {
    HealthPayload {
        status: "ok",
        service: "proxyd",
        version: env!("CARGO_PKG_VERSION"),
        active_connections: ACTIVE_CONNECTIONS.load(Ordering::Relaxed),
        total_connections: TOTAL_CONNECTIONS.load(Ordering::Relaxed),
        active_upgrades: ACTIVE_UPGRADES.load(Ordering::Relaxed),
        total_upgrades: TOTAL_UPGRADES.load(Ordering::Relaxed),
        total_requests: TOTAL_REQUESTS.load(Ordering::Relaxed),
        total_request_errors: TOTAL_REQUEST_ERRORS.load(Ordering::Relaxed),
        upstream_unavailable_total: UPSTREAM_UNAVAILABLE_TOTAL.load(Ordering::Relaxed),
        upstream_handshake_failed_total: UPSTREAM_HANDSHAKE_FAILED_TOTAL.load(Ordering::Relaxed),
        upstream_request_failed_total: UPSTREAM_REQUEST_FAILED_TOTAL.load(Ordering::Relaxed),
        upgrade_errors_total: UPGRADE_ERRORS_TOTAL.load(Ordering::Relaxed),
        relay_bytes_downstream_to_upstream: RELAY_BYTES_DOWNSTREAM_TO_UPSTREAM.load(Ordering::Relaxed),
        relay_bytes_upstream_to_downstream: RELAY_BYTES_UPSTREAM_TO_DOWNSTREAM.load(Ordering::Relaxed),
        relay_frames_downstream_to_upstream: RELAY_FRAMES_DOWNSTREAM_TO_UPSTREAM.load(Ordering::Relaxed),
        relay_frames_upstream_to_downstream: RELAY_FRAMES_UPSTREAM_TO_DOWNSTREAM.load(Ordering::Relaxed),
        relay_close_frames_total: RELAY_CLOSE_FRAMES_TOTAL.load(Ordering::Relaxed),
        relay_ping_frames_total: RELAY_PING_FRAMES_TOTAL.load(Ordering::Relaxed),
        relay_pong_frames_total: RELAY_PONG_FRAMES_TOTAL.load(Ordering::Relaxed),
    }
}

fn configured_request_timeout() -> Duration {
    let millis = std::env::var("SUB2API_RUST_REQUEST_TIMEOUT_MS")
        .ok()
        .and_then(|raw| raw.trim().parse::<u64>().ok())
        .filter(|value| *value > 0)
        .unwrap_or(30_000);
    Duration::from_millis(millis)
}

fn configured_upgrade_idle_timeout() -> Duration {
    if let Some(millis) = std::env::var("SUB2API_RUST_UPGRADE_IDLE_TIMEOUT_MS")
        .ok()
        .and_then(|raw| raw.trim().parse::<u64>().ok())
        .filter(|value| *value > 0)
    {
        return Duration::from_millis(millis);
    }
    let base = configured_request_timeout();
    if base < Duration::from_secs(120) {
        Duration::from_secs(120)
    } else {
        base
    }
}

fn configured_websocket_max_message_size() -> usize {
    std::env::var("SUB2API_RUST_WS_MAX_MESSAGE_BYTES")
        .ok()
        .and_then(|raw| raw.trim().parse::<usize>().ok())
        .filter(|value| *value > 0)
        .unwrap_or(64 << 20)
}

async fn proxy_request(mut req: Request<Incoming>, upstream_socket: &str) -> Response<ProxyBody> {
    let request_timeout = configured_request_timeout();
    let is_upgrade = is_upgrade_request(req.headers());
    let downstream_upgrade = if is_upgrade {
        Some(upgrade::on(&mut req))
    } else {
        None
    };
    TOTAL_REQUESTS.fetch_add(1, Ordering::Relaxed);

    let upstream = match timeout(request_timeout, tokio::net::UnixStream::connect(upstream_socket)).await {
        Ok(Ok(stream)) => stream,
        Ok(Err(err)) => {
            TOTAL_REQUEST_ERRORS.fetch_add(1, Ordering::Relaxed);
            UPSTREAM_UNAVAILABLE_TOTAL.fetch_add(1, Ordering::Relaxed);
            return json_error(
                StatusCode::SERVICE_UNAVAILABLE,
                &format!("upstream_unavailable: {}", err),
            );
        }
        Err(_) => {
            TOTAL_REQUEST_ERRORS.fetch_add(1, Ordering::Relaxed);
            UPSTREAM_UNAVAILABLE_TOTAL.fetch_add(1, Ordering::Relaxed);
            return json_error(StatusCode::SERVICE_UNAVAILABLE, "upstream_connect_timeout");
        }
    };

    let io = TokioIo::new(upstream);
    let (mut sender, conn) = match timeout(request_timeout, http1::handshake::<_, Incoming>(io)).await {
        Ok(Ok(parts)) => parts,
        Ok(Err(err)) => {
            TOTAL_REQUEST_ERRORS.fetch_add(1, Ordering::Relaxed);
            UPSTREAM_HANDSHAKE_FAILED_TOTAL.fetch_add(1, Ordering::Relaxed);
            return json_error(
                StatusCode::SERVICE_UNAVAILABLE,
                &format!("upstream_handshake_failed: {}", err),
            );
        }
        Err(_) => {
            TOTAL_REQUEST_ERRORS.fetch_add(1, Ordering::Relaxed);
            UPSTREAM_HANDSHAKE_FAILED_TOTAL.fetch_add(1, Ordering::Relaxed);
            return json_error(StatusCode::SERVICE_UNAVAILABLE, "upstream_handshake_timeout");
        }
    };

    tokio::spawn(async move {
        if let Err(err) = conn.with_upgrades().await {
            eprintln!("proxyd upstream connection error: {}", err);
        }
    });

    let (parts, body) = req.into_parts();
    let mut builder = Request::builder()
        .method(parts.method.clone())
        .uri(
            parts
                .uri
                .path_and_query()
                .map(|value| value.as_str())
                .unwrap_or("/"),
        )
        .version(parts.version);

    for (key, value) in &parts.headers {
        if !is_upgrade && key.as_str().eq_ignore_ascii_case("connection") {
            continue;
        }
        builder = builder.header(key, value);
    }
    builder = builder.header(BYPASS_HEADER, "1");

    let upstream_req = match builder.body(body) {
        Ok(req) => req,
        Err(err) => {
            return json_error(StatusCode::BAD_REQUEST, &format!("build_request_failed: {}", err));
        }
    };

    let mut upstream_resp = match timeout(request_timeout, sender.send_request(upstream_req)).await {
        Ok(Ok(resp)) => resp,
        Ok(Err(err)) => {
            TOTAL_REQUEST_ERRORS.fetch_add(1, Ordering::Relaxed);
            UPSTREAM_REQUEST_FAILED_TOTAL.fetch_add(1, Ordering::Relaxed);
            return json_error(StatusCode::BAD_GATEWAY, &format!("upstream_request_failed: {}", err));
        }
        Err(_) => {
            TOTAL_REQUEST_ERRORS.fetch_add(1, Ordering::Relaxed);
            UPSTREAM_REQUEST_FAILED_TOTAL.fetch_add(1, Ordering::Relaxed);
            return json_error(StatusCode::BAD_GATEWAY, "upstream_request_timeout");
        }
    };

    let upstream_upgrade = if is_upgrade && upstream_resp.status() == StatusCode::SWITCHING_PROTOCOLS {
        Some(upgrade::on(&mut upstream_resp))
    } else {
        None
    };

    let (resp_parts, resp_body) = upstream_resp.into_parts();
    let mut response = if upstream_upgrade.is_some() {
        Response::new(full_body(Bytes::new()))
    } else {
        Response::new(resp_body.boxed())
    };
    *response.status_mut() = resp_parts.status;
    *response.version_mut() = resp_parts.version;
    for (key, value) in &resp_parts.headers {
        response.headers_mut().append(key, value.clone());
    }

    if let (Some(downstream_upgrade), Some(upstream_upgrade)) = (downstream_upgrade, upstream_upgrade) {
        tokio::spawn(async move {
            ACTIVE_UPGRADES.fetch_add(1, Ordering::Relaxed);
            TOTAL_UPGRADES.fetch_add(1, Ordering::Relaxed);
            let downstream = downstream_upgrade.await;
            let upstream = upstream_upgrade.await;
            match (downstream, upstream) {
                (Ok(downstream), Ok(upstream)) => {
                    let downstream = TokioIo::new(downstream);
                    let upstream = TokioIo::new(upstream);
                    if let Err(err) = relay_upgraded_websocket_frames(downstream, upstream).await {
                        UPGRADE_ERRORS_TOTAL.fetch_add(1, Ordering::Relaxed);
                        eprintln!("proxyd relay upgrade error: {}", err);
                    }
                }
                (Err(err), _) => {
                    UPGRADE_ERRORS_TOTAL.fetch_add(1, Ordering::Relaxed);
                    eprintln!("proxyd downstream upgrade error: {}", err);
                }
                (_, Err(err)) => {
                    UPGRADE_ERRORS_TOTAL.fetch_add(1, Ordering::Relaxed);
                    eprintln!("proxyd upstream upgrade error: {}", err);
                }
            }
            ACTIVE_UPGRADES.fetch_sub(1, Ordering::Relaxed);
        });
    }

    response
}

fn json_response<T: Serialize>(status: StatusCode, payload: &T) -> Response<ProxyBody> {
    let body = match serde_json::to_vec(payload) {
        Ok(body) => body,
        Err(err) => format!(r#"{{"status":"error","message":"{}"}}"#, err).into_bytes(),
    };
    let mut response = Response::new(full_body(Bytes::from(body)));
    *response.status_mut() = status;
    response
        .headers_mut()
        .insert(CONTENT_TYPE, HeaderValue::from_static("application/json"));
    response
}

fn json_error(status: StatusCode, message: &str) -> Response<ProxyBody> {
    json_response(
        status,
        &serde_json::json!({
            "status": "error",
            "message": message,
        }),
    )
}

fn full_body(bytes: Bytes) -> ProxyBody {
    Full::new(bytes).map_err(|never| match never {}).boxed()
}

fn cleanup_socket_path(socket_path: &str) -> io::Result<()> {
    if !Path::new(socket_path).exists() {
        return Ok(());
    }
    let metadata = fs::symlink_metadata(socket_path)?;
    if metadata.file_type().is_socket() {
        fs::remove_file(socket_path)?;
        return Ok(());
    }
    Err(io::Error::new(
        io::ErrorKind::AlreadyExists,
        format!("refusing to remove non-socket path: {}", socket_path),
    ))
}

fn is_upgrade_request(headers: &hyper::HeaderMap<HeaderValue>) -> bool {
    let has_upgrade = headers
        .get(hyper::header::UPGRADE)
        .and_then(|v| v.to_str().ok())
        .map(|v| !v.trim().is_empty())
        .unwrap_or(false);
    let has_connection_upgrade = headers
        .get(hyper::header::CONNECTION)
        .and_then(|v| v.to_str().ok())
        .map(|v| v.to_ascii_lowercase().contains("upgrade"))
        .unwrap_or(false);
    has_upgrade && has_connection_upgrade
}

async fn relay_upgraded_websocket_frames<A, B>(downstream: A, upstream: B) -> Result<(), WebSocketError>
where
    A: tokio::io::AsyncRead + tokio::io::AsyncWrite + Unpin + Send + 'static,
    B: tokio::io::AsyncRead + tokio::io::AsyncWrite + Unpin + Send + 'static,
{
    let (downstream_read, downstream_write) = tokio::io::split(downstream);
    let (upstream_read, upstream_write) = tokio::io::split(upstream);

    let (mut downstream_read, mut downstream_write) =
        after_handshake_split(downstream_read, downstream_write, Role::Server);
    let (mut upstream_read, mut upstream_write) =
        after_handshake_split(upstream_read, upstream_write, Role::Client);

    downstream_read.set_auto_close(false);
    downstream_read.set_auto_pong(false);
    downstream_read.set_max_message_size(configured_websocket_max_message_size());
    upstream_read.set_auto_close(false);
    upstream_read.set_auto_pong(false);
    upstream_read.set_max_message_size(configured_websocket_max_message_size());
    let idle_timeout = configured_upgrade_idle_timeout();

    let downstream_to_upstream = tokio::spawn(async move {
        relay_websocket_frames(
            &mut downstream_read,
            &mut upstream_write,
            &RELAY_BYTES_DOWNSTREAM_TO_UPSTREAM,
            &RELAY_FRAMES_DOWNSTREAM_TO_UPSTREAM,
            idle_timeout,
        )
        .await
    });
    let upstream_to_downstream = tokio::spawn(async move {
        relay_websocket_frames(
            &mut upstream_read,
            &mut downstream_write,
            &RELAY_BYTES_UPSTREAM_TO_DOWNSTREAM,
            &RELAY_FRAMES_UPSTREAM_TO_DOWNSTREAM,
            idle_timeout,
        )
        .await
    });

    wait_for_relay_pair(downstream_to_upstream, upstream_to_downstream).await
}

async fn relay_websocket_frames<R, W>(
    reader: &mut fastwebsockets::WebSocketRead<R>,
    writer: &mut fastwebsockets::WebSocketWrite<W>,
    byte_counter: &AtomicI64,
    frame_counter: &AtomicI64,
    idle_timeout: Duration,
) -> Result<(), WebSocketError>
where
    R: tokio::io::AsyncRead + Unpin,
    W: tokio::io::AsyncWrite + Unpin,
{
    loop {
        let frame = timeout(
            idle_timeout,
            reader.read_frame(&mut |_| async { Ok::<(), WebSocketError>(()) }),
        )
        .await
        .map_err(|_| {
            WebSocketError::SendError(
                io::Error::new(io::ErrorKind::TimedOut, "websocket frame read timeout").into(),
            )
        })??;
        let opcode = frame.opcode;
        add_u64_counter(byte_counter, frame.payload.len() as u64);
        frame_counter.fetch_add(1, Ordering::Relaxed);
        match opcode {
            OpCode::Close => {
                RELAY_CLOSE_FRAMES_TOTAL.fetch_add(1, Ordering::Relaxed);
            }
            OpCode::Ping => {
                RELAY_PING_FRAMES_TOTAL.fetch_add(1, Ordering::Relaxed);
            }
            OpCode::Pong => {
                RELAY_PONG_FRAMES_TOTAL.fetch_add(1, Ordering::Relaxed);
            }
            _ => {}
        }
        timeout(idle_timeout, writer.write_frame(frame))
            .await
            .map_err(|_| {
                WebSocketError::SendError(
                    io::Error::new(io::ErrorKind::TimedOut, "websocket frame write timeout").into(),
                )
            })??;
        if opcode == OpCode::Close {
            break;
        }
    }
    Ok(())
}

async fn wait_for_relay_pair(
    mut left: tokio::task::JoinHandle<Result<(), WebSocketError>>,
    mut right: tokio::task::JoinHandle<Result<(), WebSocketError>>,
) -> Result<(), WebSocketError> {
    let (left_finished, first_result) = tokio::select! {
        result = &mut left => (true, result),
        result = &mut right => (false, result),
    };

    let first_result = first_result.map_err(|err| WebSocketError::SendError(err.into()))?;

    let drain_timeout = Duration::from_secs(1);
    if left_finished {
        match timeout(drain_timeout, &mut right).await {
            Ok(result) => {
                result.map_err(|err| WebSocketError::SendError(err.into()))??;
            }
            Err(_) => right.abort(),
        }
    } else {
        match timeout(drain_timeout, &mut left).await {
            Ok(result) => {
                result.map_err(|err| WebSocketError::SendError(err.into()))??;
            }
            Err(_) => left.abort(),
        }
    }

    first_result
}

fn add_u64_counter(counter: &AtomicI64, value: u64) {
    let delta = i64::try_from(value).unwrap_or(i64::MAX);
    counter.fetch_add(delta, Ordering::Relaxed);
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::{Mutex, OnceLock};
    use std::os::unix::net::UnixListener as StdUnixListener;
    use std::time::{SystemTime, UNIX_EPOCH};

    use hyper::client::conn::http1;
    use tokio::io::{AsyncReadExt, AsyncWriteExt};

    fn unique_socket_path(prefix: &str) -> String {
        let nanos = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("time")
            .as_nanos();
        std::env::temp_dir()
            .join(format!("{}-{}-{}.sock", prefix, std::process::id(), nanos))
            .to_string_lossy()
            .to_string()
    }

    fn env_test_lock() -> std::sync::MutexGuard<'static, ()> {
        static ENV_LOCK: OnceLock<Mutex<()>> = OnceLock::new();
        ENV_LOCK
            .get_or_init(|| Mutex::new(()))
            .lock()
            .expect("lock env test mutex")
    }

    async fn write_ws_frame<W>(writer: &mut W, opcode: u8, payload: &[u8], masked: bool)
    where
        W: tokio::io::AsyncWrite + Unpin,
    {
        let mut frame = Vec::with_capacity(payload.len() + 14);
        frame.push(0x80 | (opcode & 0x0f));

        let payload_len = payload.len();
        let mask_bit = if masked { 0x80 } else { 0x00 };
        if payload_len < 126 {
            frame.push(mask_bit | payload_len as u8);
        } else if payload_len <= u16::MAX as usize {
            frame.push(mask_bit | 126);
            frame.extend_from_slice(&(payload_len as u16).to_be_bytes());
        } else {
            frame.push(mask_bit | 127);
            frame.extend_from_slice(&(payload_len as u64).to_be_bytes());
        }

        if masked {
            let mask = [0x11_u8, 0x22_u8, 0x33_u8, 0x44_u8];
            frame.extend_from_slice(&mask);
            for (idx, b) in payload.iter().enumerate() {
                frame.push(*b ^ mask[idx % 4]);
            }
        } else {
            frame.extend_from_slice(payload);
        }

        writer.write_all(&frame).await.expect("write websocket frame");
    }

    async fn read_ws_frame<R>(reader: &mut R) -> (u8, Vec<u8>)
    where
        R: tokio::io::AsyncRead + Unpin,
    {
        let mut header = [0u8; 2];
        reader.read_exact(&mut header).await.expect("read frame header");

        let opcode = header[0] & 0x0f;
        let masked = header[1]&0x80 != 0;
        let mut payload_len = usize::from(header[1] & 0x7f);
        if payload_len == 126 {
            let mut extended = [0u8; 2];
            reader
                .read_exact(&mut extended)
                .await
                .expect("read extended payload len");
            payload_len = usize::from(u16::from_be_bytes(extended));
        } else if payload_len == 127 {
            let mut extended = [0u8; 8];
            reader
                .read_exact(&mut extended)
                .await
                .expect("read extended payload len");
            payload_len = usize::try_from(u64::from_be_bytes(extended)).expect("payload len fits usize");
        }

        let mut mask = [0u8; 4];
        if masked {
            reader.read_exact(&mut mask).await.expect("read mask");
        }

        let mut payload = vec![0u8; payload_len];
        reader.read_exact(&mut payload).await.expect("read payload");
        if masked {
            for (idx, b) in payload.iter_mut().enumerate() {
                *b ^= mask[idx % 4];
            }
        }
        (opcode, payload)
    }

    #[tokio::test]
    async fn proxies_request_to_unix_upstream() {
        let upstream_socket = unique_socket_path("sub2api-upstream");
        let proxyd_socket = unique_socket_path("sub2api-proxyd");

        let upstream_listener = UnixListener::bind(&upstream_socket).expect("bind upstream");
        let proxyd_listener = UnixListener::bind(&proxyd_socket).expect("bind proxyd");

        let upstream_task = tokio::spawn(async move {
            let (stream, _) = upstream_listener.accept().await.expect("accept upstream");
            let io = TokioIo::new(stream);
            let svc = service_fn(|req: Request<Incoming>| async move {
                let body = req.into_body().collect().await.expect("collect req").to_bytes();
                let mut resp = Response::new(full_body(body));
                *resp.status_mut() = StatusCode::CREATED;
                resp.headers_mut()
                    .insert(hyper::header::CONNECTION, HeaderValue::from_static("close"));
                Ok::<_, Infallible>(resp)
            });
            let builder = AutoBuilder::new(TokioExecutor::new());
            builder
                .serve_connection_with_upgrades(io, svc)
                .await
                .expect("serve upstream");
        });

        let upstream_socket_for_proxy = upstream_socket.clone();
        let proxyd_task = tokio::spawn(async move {
            let (stream, _) = proxyd_listener.accept().await.expect("accept proxyd");
            let io = TokioIo::new(stream);
            let svc = service_fn(move |req| {
                let upstream_socket = upstream_socket_for_proxy.clone();
                async move { Ok::<_, Infallible>(handle_request_with_upstream(req, upstream_socket).await) }
            });
            let builder = AutoBuilder::new(TokioExecutor::new());
            builder
                .serve_connection_with_upgrades(io, svc)
                .await
                .expect("serve proxyd");
        });

        let client_stream = tokio::net::UnixStream::connect(&proxyd_socket)
            .await
            .expect("connect proxyd");
        let io = TokioIo::new(client_stream);
        let (mut sender, conn) = http1::handshake::<_, Full<Bytes>>(io)
            .await
            .expect("client handshake");
        let conn_task = tokio::spawn(async move {
            conn.await.expect("client conn");
        });

        let req = Request::builder()
            .method(Method::POST)
            .uri("/echo")
            .header(hyper::header::CONNECTION, "close")
            .body(Full::new(Bytes::from_static(b"hello-sidecar")))
            .expect("build req");
        let resp = sender.send_request(req).await.expect("send request");
        let status = resp.status();
        let body = resp.into_body().collect().await.expect("collect resp").to_bytes();

        assert_eq!(status, StatusCode::CREATED);
        assert_eq!(body.as_ref(), b"hello-sidecar");

        drop(sender);
        conn_task.await.expect("client join");
        proxyd_task.await.expect("proxyd join");
        upstream_task.await.expect("upstream join");

        let _ = fs::remove_file(&proxyd_socket);
        let _ = fs::remove_file(&upstream_socket);
    }

    #[tokio::test]
    async fn proxies_upgrade_tunnel_to_unix_upstream() {
        let upstream_socket = unique_socket_path("sub2api-upstream-upgrade");
        let proxyd_socket = unique_socket_path("sub2api-proxyd-upgrade");

        let upstream_listener = UnixListener::bind(&upstream_socket).expect("bind upstream");
        let proxyd_listener = UnixListener::bind(&proxyd_socket).expect("bind proxyd");

        let upstream_task = tokio::spawn(async move {
            let (stream, _) = upstream_listener.accept().await.expect("accept upstream");
            let io = TokioIo::new(stream);
            let svc = service_fn(|mut req: Request<Incoming>| async move {
                let on_upgrade = upgrade::on(&mut req);
                tokio::spawn(async move {
                    let upgraded = on_upgrade.await.expect("upstream upgrade");
                    let mut upgraded = TokioIo::new(upgraded);
                    let (opcode, payload) = read_ws_frame(&mut upgraded).await;
                    assert_eq!(opcode, 0x2);
                    assert_eq!(payload.as_slice(), b"pong");
                    write_ws_frame(&mut upgraded, 0x2, &payload, false).await;

                    let (close_opcode, _) = read_ws_frame(&mut upgraded).await;
                    assert_eq!(close_opcode, 0x8);
                    write_ws_frame(&mut upgraded, 0x8, &[], false).await;
                });

                let mut resp = Response::new(full_body(Bytes::new()));
                *resp.status_mut() = StatusCode::SWITCHING_PROTOCOLS;
                resp.headers_mut()
                    .insert(hyper::header::CONNECTION, HeaderValue::from_static("Upgrade"));
                resp.headers_mut()
                    .insert(hyper::header::UPGRADE, HeaderValue::from_static("websocket"));
                Ok::<_, Infallible>(resp)
            });
            let builder = AutoBuilder::new(TokioExecutor::new());
            builder
                .serve_connection_with_upgrades(io, svc)
                .await
                .expect("serve upstream");
        });

        let upstream_socket_for_proxy = upstream_socket.clone();
        let proxyd_task = tokio::spawn(async move {
            let (stream, _) = proxyd_listener.accept().await.expect("accept proxyd");
            let io = TokioIo::new(stream);
            let svc = service_fn(move |req| {
                let upstream_socket = upstream_socket_for_proxy.clone();
                async move { Ok::<_, Infallible>(handle_request_with_upstream(req, upstream_socket).await) }
            });
            let builder = AutoBuilder::new(TokioExecutor::new());
            builder
                .serve_connection_with_upgrades(io, svc)
                .await
                .expect("serve proxyd");
        });

        let mut client = tokio::net::UnixStream::connect(&proxyd_socket)
            .await
            .expect("connect proxyd");
        client
            .write_all(
                b"GET /v1/responses HTTP/1.1\r\nHost: example.com\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: test-key\r\nSec-WebSocket-Version: 13\r\n\r\n",
            )
            .await
            .expect("write upgrade request");

        let mut resp_bytes = Vec::new();
        let mut buf = [0u8; 1];
        loop {
            client.read_exact(&mut buf).await.expect("read upgrade response");
            resp_bytes.push(buf[0]);
            if resp_bytes.ends_with(b"\r\n\r\n") {
                break;
            }
        }
        let response_text = String::from_utf8(resp_bytes).expect("utf8");
        assert!(response_text.contains("101 Switching Protocols"));

        write_ws_frame(&mut client, 0x2, b"pong", true).await;
        let (echo_opcode, echoed) = read_ws_frame(&mut client).await;
        assert_eq!(echo_opcode, 0x2);
        assert_eq!(echoed.as_slice(), b"pong");
        write_ws_frame(&mut client, 0x8, &[], true).await;
        let (close_opcode, _) = read_ws_frame(&mut client).await;
        assert_eq!(close_opcode, 0x8);
        drop(client);

        proxyd_task.await.expect("proxyd join");
        upstream_task.await.expect("upstream join");

        let _ = fs::remove_file(&proxyd_socket);
        let _ = fs::remove_file(&upstream_socket);
    }

    #[tokio::test]
    async fn health_response_includes_runtime_counters() {
        ACTIVE_CONNECTIONS.store(2, Ordering::Relaxed);
        TOTAL_CONNECTIONS.store(3, Ordering::Relaxed);
        ACTIVE_UPGRADES.store(1, Ordering::Relaxed);
        TOTAL_UPGRADES.store(4, Ordering::Relaxed);
        TOTAL_REQUESTS.store(5, Ordering::Relaxed);
        TOTAL_REQUEST_ERRORS.store(1, Ordering::Relaxed);
        UPSTREAM_UNAVAILABLE_TOTAL.store(2, Ordering::Relaxed);
        UPSTREAM_HANDSHAKE_FAILED_TOTAL.store(3, Ordering::Relaxed);
        UPSTREAM_REQUEST_FAILED_TOTAL.store(4, Ordering::Relaxed);
        UPGRADE_ERRORS_TOTAL.store(5, Ordering::Relaxed);
        RELAY_BYTES_DOWNSTREAM_TO_UPSTREAM.store(6, Ordering::Relaxed);
        RELAY_BYTES_UPSTREAM_TO_DOWNSTREAM.store(7, Ordering::Relaxed);
        RELAY_FRAMES_DOWNSTREAM_TO_UPSTREAM.store(8, Ordering::Relaxed);
        RELAY_FRAMES_UPSTREAM_TO_DOWNSTREAM.store(9, Ordering::Relaxed);
        RELAY_CLOSE_FRAMES_TOTAL.store(10, Ordering::Relaxed);
        RELAY_PING_FRAMES_TOTAL.store(11, Ordering::Relaxed);
        RELAY_PONG_FRAMES_TOTAL.store(12, Ordering::Relaxed);

        let payload = serde_json::to_value(current_health_payload()).expect("serialize health");
        assert_eq!(payload.get("active_connections").and_then(serde_json::Value::as_i64), Some(2));
        assert_eq!(payload.get("total_connections").and_then(serde_json::Value::as_i64), Some(3));
        assert_eq!(payload.get("active_upgrades").and_then(serde_json::Value::as_i64), Some(1));
        assert_eq!(payload.get("total_upgrades").and_then(serde_json::Value::as_i64), Some(4));
        assert_eq!(payload.get("total_requests").and_then(serde_json::Value::as_i64), Some(5));
        assert_eq!(payload.get("total_request_errors").and_then(serde_json::Value::as_i64), Some(1));
        assert_eq!(payload.get("upstream_unavailable_total").and_then(serde_json::Value::as_i64), Some(2));
        assert_eq!(payload.get("upstream_handshake_failed_total").and_then(serde_json::Value::as_i64), Some(3));
        assert_eq!(payload.get("upstream_request_failed_total").and_then(serde_json::Value::as_i64), Some(4));
        assert_eq!(payload.get("upgrade_errors_total").and_then(serde_json::Value::as_i64), Some(5));
        assert_eq!(payload.get("relay_bytes_downstream_to_upstream").and_then(serde_json::Value::as_i64), Some(6));
        assert_eq!(payload.get("relay_bytes_upstream_to_downstream").and_then(serde_json::Value::as_i64), Some(7));
        assert_eq!(payload.get("relay_frames_downstream_to_upstream").and_then(serde_json::Value::as_i64), Some(8));
        assert_eq!(payload.get("relay_frames_upstream_to_downstream").and_then(serde_json::Value::as_i64), Some(9));
        assert_eq!(payload.get("relay_close_frames_total").and_then(serde_json::Value::as_i64), Some(10));
        assert_eq!(payload.get("relay_ping_frames_total").and_then(serde_json::Value::as_i64), Some(11));
        assert_eq!(payload.get("relay_pong_frames_total").and_then(serde_json::Value::as_i64), Some(12));
    }

    #[test]
    fn configured_request_timeout_uses_env_override() {
        let _guard = env_test_lock();
        std::env::set_var("SUB2API_RUST_REQUEST_TIMEOUT_MS", "1234");
        assert_eq!(configured_request_timeout(), Duration::from_millis(1234));
        std::env::remove_var("SUB2API_RUST_REQUEST_TIMEOUT_MS");
    }

    #[test]
    fn configured_upgrade_idle_timeout_has_floor() {
        let _guard = env_test_lock();
        std::env::set_var("SUB2API_RUST_REQUEST_TIMEOUT_MS", "1234");
        assert_eq!(configured_upgrade_idle_timeout(), Duration::from_secs(120));
        std::env::set_var("SUB2API_RUST_UPGRADE_IDLE_TIMEOUT_MS", "9000");
        assert_eq!(configured_upgrade_idle_timeout(), Duration::from_millis(9000));
        std::env::remove_var("SUB2API_RUST_UPGRADE_IDLE_TIMEOUT_MS");
        std::env::set_var("SUB2API_RUST_REQUEST_TIMEOUT_MS", "240000");
        assert_eq!(configured_upgrade_idle_timeout(), Duration::from_millis(240000));
        std::env::remove_var("SUB2API_RUST_REQUEST_TIMEOUT_MS");
    }

    #[test]
    fn configured_websocket_max_message_size_uses_env_override() {
        let _guard = env_test_lock();
        std::env::set_var("SUB2API_RUST_WS_MAX_MESSAGE_BYTES", "8192");
        assert_eq!(configured_websocket_max_message_size(), 8192);
        std::env::remove_var("SUB2API_RUST_WS_MAX_MESSAGE_BYTES");
        assert_eq!(configured_websocket_max_message_size(), 64 << 20);
    }

    #[tokio::test]
    async fn proxy_request_times_out_when_upstream_never_responds() {
        let _guard = env_test_lock();
        let upstream_socket = unique_socket_path("sub2api-upstream-timeout");
        let proxyd_socket = unique_socket_path("sub2api-proxyd-timeout");

        let upstream_listener = UnixListener::bind(&upstream_socket).expect("bind upstream");
        let proxyd_listener = UnixListener::bind(&proxyd_socket).expect("bind proxyd");

        let upstream_task = tokio::spawn(async move {
            let (_stream, _) = upstream_listener.accept().await.expect("accept upstream");
            tokio::time::sleep(Duration::from_millis(200)).await;
        });

        let upstream_socket_for_proxy = upstream_socket.clone();
        let proxyd_task = tokio::spawn(async move {
            let (stream, _) = proxyd_listener.accept().await.expect("accept proxyd");
            let io = TokioIo::new(stream);
            let svc = service_fn(move |req| {
                let upstream_socket = upstream_socket_for_proxy.clone();
                async move { Ok::<_, Infallible>(handle_request_with_upstream(req, upstream_socket).await) }
            });
            let builder = AutoBuilder::new(TokioExecutor::new());
            builder
                .serve_connection_with_upgrades(io, svc)
                .await
                .expect("serve proxyd");
        });

        std::env::set_var("SUB2API_RUST_REQUEST_TIMEOUT_MS", "50");
        let client_stream = tokio::net::UnixStream::connect(&proxyd_socket)
            .await
            .expect("connect proxyd");
        let io = TokioIo::new(client_stream);
        let (mut sender, conn) = http1::handshake::<_, Full<Bytes>>(io)
            .await
            .expect("client handshake");
        let conn_task = tokio::spawn(async move {
            let _ = conn.await;
        });

        let req = Request::builder()
            .method(Method::POST)
            .uri("/echo")
            .header(hyper::header::CONNECTION, "close")
            .body(Full::new(Bytes::from_static(b"hello-timeout")))
            .expect("build req");
        let resp = sender.send_request(req).await.expect("send request");
        let status = resp.status();
        let body = resp.into_body().collect().await.expect("collect body").to_bytes();
        let payload: serde_json::Value = serde_json::from_slice(&body).expect("parse body");

        assert_eq!(status, StatusCode::BAD_GATEWAY);
        assert_eq!(payload.get("message").and_then(serde_json::Value::as_str), Some("upstream_request_timeout"));

        drop(sender);
        conn_task.await.expect("conn join");
        proxyd_task.await.expect("proxyd join");
        upstream_task.await.expect("upstream join");
        std::env::remove_var("SUB2API_RUST_REQUEST_TIMEOUT_MS");
        let _ = fs::remove_file(&proxyd_socket);
        let _ = fs::remove_file(&upstream_socket);
    }

    #[test]
    fn cleanup_socket_path_removes_existing_socket() {
        let socket_path = unique_socket_path("sub2api-cleanup-socket");
        let listener = StdUnixListener::bind(&socket_path).expect("bind socket");
        drop(listener);

        cleanup_socket_path(&socket_path).expect("cleanup socket path");
        assert!(!Path::new(&socket_path).exists());
    }

    #[test]
    fn cleanup_socket_path_rejects_regular_file() {
        let file_path = unique_socket_path("sub2api-cleanup-file");
        fs::write(&file_path, b"not-a-socket").expect("write file");
        let err = cleanup_socket_path(&file_path).expect_err("should reject regular file");
        assert_eq!(err.kind(), io::ErrorKind::AlreadyExists);
        let _ = fs::remove_file(&file_path);
    }
}
