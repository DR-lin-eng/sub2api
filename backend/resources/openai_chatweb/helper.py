import json
import sys

from sentinel_go_port import (
    Persona,
    Session,
    SentinelSDK,
    solve_session_observer_with_proof,
    solve_turnstile_dx_with_session,
)


def _build_session(payload):
    persona_payload = payload.get("persona") or {}
    persona = Persona(
        platform=str(persona_payload.get("platform") or ""),
        vendor=str(persona_payload.get("vendor") or ""),
        timezone_offset_min=int(persona_payload.get("timezone_offset_min") or 0),
        session_id=str(persona_payload.get("session_id") or ""),
        time_origin=float(persona_payload.get("time_origin") or 0.0),
        window_flags=[int(item or 0) for item in (persona_payload.get("window_flags") or [])],
        window_flags_set=bool(persona_payload.get("window_flags_set")),
        entropy_a=float(persona_payload.get("entropy_a") or 0.0),
        entropy_b=float(persona_payload.get("entropy_b") or 0.0),
        date_string=str(persona_payload.get("date_string") or ""),
        requirements_script_url=str(persona_payload.get("requirements_script_url") or ""),
        navigator_probe=str(persona_payload.get("navigator_probe") or ""),
        document_probe=str(persona_payload.get("document_probe") or ""),
        window_probe=str(persona_payload.get("window_probe") or ""),
        performance_now=float(persona_payload.get("performance_now") or 0.0),
        requirements_elapsed=float(persona_payload.get("requirements_elapsed") or 0.0),
    )
    return Session(
        device_id=str(payload.get("device_id") or ""),
        user_agent=str(payload.get("user_agent") or ""),
        screen_width=int(payload.get("screen_width") or 0),
        screen_height=int(payload.get("screen_height") or 0),
        heap_limit=int(payload.get("heap_limit") or 0),
        hardware_concurrency=int(payload.get("hardware_concurrency") or 0),
        language=str(payload.get("language") or ""),
        languages_join=str(payload.get("languages_join") or ""),
        persona=persona,
    )


def _run(op, payload):
    session = _build_session(payload.get("session") or {})
    sdk = SentinelSDK(session)

    if op == "requirements_token":
        return {"requirements_token": sdk.requirements_token()}
    if op == "enforcement_token":
        return {
            "proof_token": sdk.enforcement_token(
                bool(payload.get("required")),
                str(payload.get("seed") or ""),
                str(payload.get("difficulty") or ""),
            )
        }
    if op == "solve_turnstile_dx":
        return {
            "turnstile_token": solve_turnstile_dx_with_session(
                str(payload.get("requirements_token") or ""),
                str(payload.get("dx") or ""),
                session,
            )
        }
    if op == "solve_session_observer":
        return {
            "so_token": solve_session_observer_with_proof(
                str(payload.get("proof_token") or ""),
                str(payload.get("collector_dx") or ""),
                session,
            )
        }
    raise ValueError(f"unsupported op: {op}")


def main():
    payload = json.loads(sys.stdin.read() or "{}")
    op = str(payload.get("op") or "").strip()
    if not op:
        raise SystemExit("missing op")
    result = _run(op, payload)
    sys.stdout.write(json.dumps(result, ensure_ascii=False))


if __name__ == "__main__":
    main()
