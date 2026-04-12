from __future__ import annotations

import base64
import json
import math
import random
import re
import secrets
import string
import time
from dataclasses import dataclass, field
from datetime import datetime, timedelta, timezone
from typing import Any, Callable


TURNSTILE_QUEUE_REG = 9
TURNSTILE_WINDOW_REG = 10
TURNSTILE_KEY_REG = 16
TURNSTILE_SUCCESS_REG = 3
TURNSTILE_ERROR_REG = 4
TURNSTILE_CALLBACK_REG = 30
ORDERED_KEYS_META = "__ordered_keys__"
PROTOTYPE_META = "__prototype__"
DEFAULT_SENTINEL_SDK_URL = "https://sentinel.openai.com/sentinel/20260219f9f6/sdk.js"

_SEEDED = random.Random(time.time_ns())


@dataclass
class Persona:
    platform: str = ""
    vendor: str = ""
    timezone_offset_min: int = 0
    session_id: str = ""
    time_origin: float = 0.0
    window_flags: list[int] = field(default_factory=list)
    window_flags_set: bool = False
    entropy_a: float = 0.0
    entropy_b: float = 0.0
    date_string: str = ""
    requirements_script_url: str = ""
    navigator_probe: str = ""
    document_probe: str = ""
    window_probe: str = ""
    performance_now: float = 0.0
    requirements_elapsed: float = 0.0


@dataclass
class Session:
    device_id: str = ""
    user_agent: str = ""
    screen_width: int = 1920
    screen_height: int = 1080
    heap_limit: int = 4294967296
    hardware_concurrency: int = 8
    language: str = "en-US"
    languages_join: str = "en-US,en"
    persona: Persona = field(default_factory=Persona)


def random_hex(n: int) -> str:
    if n <= 0:
        return ""
    return secrets.token_hex((n + 1) // 2)[:n]


def random_choice(values: list[str], fallback: str) -> str:
    if not values:
        return fallback
    return values[_SEEDED.randint(0, len(values) - 1)]


def random_int(min_value: int, max_value: int) -> int:
    if max_value <= min_value:
        return min_value
    return _SEEDED.randint(min_value, max_value)


def browser_entropy_fallback() -> float:
    return float(random_int(10000, 99999)) / 100000.0


def must_b64_json(value: Any) -> str:
    body = json.dumps(value, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
    return base64.b64encode(body).decode("ascii")


def mixed_fnv(text: str) -> str:
    value = 2166136261
    for ch in text:
        value ^= ord(ch)
        value = (value * 16777619) & 0xFFFFFFFF
    value ^= value >> 16
    value = (value * 2246822507) & 0xFFFFFFFF
    value ^= value >> 13
    value = (value * 3266489909) & 0xFFFFFFFF
    value ^= value >> 16
    return format(value & 0xFFFFFFFF, "08x")


def localized_timezone_name(timezone_offset_min: int, language: str) -> str:
    lang = (language or "").strip().lower()
    if timezone_offset_min == 420:
        return "北美山区标准时间" if lang.startswith("zh") else "Mountain Standard Time"
    if timezone_offset_min == 480:
        return "太平洋标准时间" if lang.startswith("zh") else "Pacific Standard Time"
    if timezone_offset_min == 0:
        return "协调世界时" if lang.startswith("zh") else "Coordinated Universal Time"
    return ""


def sentinel_probe_defaults(language: str) -> tuple[str, str, str]:
    if (language or "").strip().lower().startswith("zh"):
        return ("clipboard−[object Clipboard]", "__reactContainer$b63yiita51i", "releaseEvents")
    return ("xr−[object XRSystem]", "location", "ondblclick")


def split_screen_sum(total: int) -> tuple[int, int]:
    common = [
        (2048, 1152),
        (1920, 1080),
        (1536, 864),
        (1440, 900),
        (1600, 900),
        (1366, 768),
    ]
    for width, height in common:
        if width + height == total:
            return width, height
    if total > 2000:
        width = round(total * 0.64)
        return int(width), int(total - width)
    return total, 0


class SentinelSDK:
    def __init__(self, session: Session | None):
        persona = session.persona if session else Persona()
        flags = [[0, 0, 0, 0, 0, 0, 0], [1, 0, 0, 0, 0, 0, 0], [0, 1, 0, 0, 0, 0, 0]]
        flag_choice = flags[random_int(0, len(flags) - 1)]
        width = 1920
        height = 1080
        heap_limit = 4294967296
        hardware_concurrency = 8
        language = "en-US"
        languages_join = "en-US,en"
        platform = "Win32"
        vendor = "Google Inc."
        timezone_offset_min = 0
        device_id = ""
        user_agent = ""
        session_id = random_hex(32)
        time_origin = float(int((time.time() - random_int(5, 120)) * 1000))
        entropy_a = browser_entropy_fallback()
        entropy_b = browser_entropy_fallback()
        date_string_override = ""
        script_url = DEFAULT_SENTINEL_SDK_URL
        navigator_probe_value, document_probe_value, window_probe_value = sentinel_probe_defaults(language)
        performance_now_value = 0.0
        requirements_elapsed = 0.0

        if session:
            device_id = session.device_id
            user_agent = session.user_agent
            if session.screen_width > 0:
                width = session.screen_width
            if session.screen_height > 0:
                height = session.screen_height
            if session.heap_limit > 0:
                heap_limit = session.heap_limit
            if session.hardware_concurrency > 0:
                hardware_concurrency = session.hardware_concurrency
            if (session.language or "").strip():
                language = session.language
            if (session.languages_join or "").strip():
                languages_join = session.languages_join
            if (persona.platform or "").strip():
                platform = persona.platform
            if (persona.vendor or "").strip():
                vendor = persona.vendor
            if persona.timezone_offset_min != 0:
                timezone_offset_min = persona.timezone_offset_min
            if persona.session_id:
                session_id = persona.session_id
            if persona.time_origin > 0:
                time_origin = persona.time_origin
            if persona.window_flags_set or persona.window_flags:
                if len(persona.window_flags) == 7:
                    flag_choice = [int(v) for v in persona.window_flags]
            if persona.entropy_a > 0:
                entropy_a = persona.entropy_a
            if persona.entropy_b > 0:
                entropy_b = persona.entropy_b
            if (persona.date_string or "").strip():
                date_string_override = persona.date_string.strip()
            if (persona.requirements_script_url or "").strip():
                script_url = persona.requirements_script_url.strip()
            if (persona.navigator_probe or "").strip():
                navigator_probe_value = persona.navigator_probe.strip()
            if (persona.document_probe or "").strip():
                document_probe_value = persona.document_probe.strip()
            if (persona.window_probe or "").strip():
                window_probe_value = persona.window_probe.strip()
            if persona.performance_now > 0:
                performance_now_value = persona.performance_now
            if persona.requirements_elapsed > 0:
                requirements_elapsed = persona.requirements_elapsed

        self.device_id = device_id
        self.session_id = session_id
        self.user_agent = user_agent
        self.screen_width = width
        self.screen_height = height
        self.heap_limit = heap_limit
        self.hardware_concurrency = hardware_concurrency
        self.time_origin = time_origin
        self.perf_start = time.monotonic()
        self.language = language
        self.languages_join = languages_join
        self.platform = platform
        self.vendor = vendor
        self.timezone_offset_min = timezone_offset_min
        self.window_flags = flag_choice
        self.entropy_a = entropy_a
        self.entropy_b = entropy_b
        self.date_string_override = date_string_override
        self.script_url = script_url
        self.navigator_probe_value = navigator_probe_value
        self.document_probe_value = document_probe_value
        self.window_probe_value = window_probe_value
        self.performance_now_value = performance_now_value
        self.requirements_elapsed = requirements_elapsed

    def perf_now(self) -> float:
        if self.performance_now_value > 0:
            return self.performance_now_value
        return (time.monotonic() - self.perf_start) * 1000.0

    def requirements_elapsed_now(self) -> float:
        if self.requirements_elapsed > 0:
            return self.requirements_elapsed
        return (time.monotonic() - self.perf_start) * 1000.0

    def date_string(self) -> str:
        if self.date_string_override:
            return self.date_string_override
        offset_minutes = -self.timezone_offset_min
        sign = "+"
        if offset_minutes < 0:
            sign = "-"
            offset_minutes = -offset_minutes
        hours = offset_minutes // 60
        minutes = offset_minutes % 60
        delta = timedelta(hours=hours, minutes=minutes)
        if sign == "-":
            delta = -delta
        local_now = datetime.now(timezone.utc) + delta
        label = localized_timezone_name(self.timezone_offset_min, self.language)
        if label:
            return local_now.strftime(f"%a %b %d %Y %H:%M:%S GMT{sign}{hours:02d}{minutes:02d} ({label})")
        return local_now.strftime(f"%a %b %d %Y %H:%M:%S GMT{sign}{hours:02d}{minutes:02d}")

    def navigator_probe(self) -> str:
        if self.navigator_probe_value:
            return self.navigator_probe_value
        probes = [
            f"hardwareConcurrency−{max(1, self.hardware_concurrency)}",
            "language−" + self.language.strip(),
            "languages−" + self.languages_join.strip(),
            "platform−" + self.platform.strip(),
        ]
        if self.vendor.strip():
            probes.append("vendor−" + self.vendor.strip())
        return random_choice(probes, probes[0])

    def document_probe(self) -> str:
        if self.document_probe_value:
            return self.document_probe_value
        session_suffix = (self.session_id or "").strip().lower()[:11] or random_hex(11)
        probes = [f"__reactContainer${session_suffix}", "onvisibilitychange", "hidden", "readyState", "characterSet"]
        return random_choice(probes, probes[0])

    def window_probe(self) -> str:
        if self.window_probe_value:
            return self.window_probe_value
        probes = ["__oai_so_bm", "ondragend", "onbeforematch", "__next_f", "__oai_cached_session"]
        return random_choice(probes, probes[0])

    def fingerprint_config(self, for_pow: bool, nonce: int, elapsed: int) -> list[Any]:
        field3: Any = self.entropy_a
        field9: Any = self.entropy_b
        if for_pow:
            field3 = nonce
            field9 = elapsed
        return [
            self.screen_width + self.screen_height,
            self.date_string(),
            self.heap_limit,
            field3,
            self.user_agent,
            self.script_url,
            None,
            self.language,
            self.languages_join,
            field9,
            self.navigator_probe(),
            self.document_probe(),
            self.window_probe(),
            self.perf_now(),
            self.session_id,
            "",
            self.hardware_concurrency,
            self.time_origin,
            *self.window_flags,
        ]

    def requirements_token(self) -> str:
        cfg = self.fingerprint_config(False, 1, 0)
        cfg[3] = 1
        cfg[9] = self.requirements_elapsed_now()
        return "gAAAAAC" + must_b64_json(cfg) + "~S"

    def enforcement_token(self, required: bool, seed: str, difficulty: str) -> str:
        if required:
            answer, _ = self.solve(seed, difficulty)
            return "gAAAAAB" + answer
        return "gAAAAAB" + must_b64_json(self.fingerprint_config(False, 0, 0)) + "~S"

    def solve(self, seed: str, difficulty: str) -> tuple[str, int]:
        difficulty = difficulty or "0"
        cfg = self.fingerprint_config(True, 0, 0)
        started = time.monotonic()
        for nonce in range(500000):
            cfg[3] = nonce
            cfg[9] = int((time.monotonic() - started) * 1000)
            answer = must_b64_json(cfg)
            if mixed_fnv(seed + answer)[: len(difficulty)] <= difficulty:
                return answer + "~S", nonce + 1
        return "wQ8Lk5FbGpA2NcR9dShT6gYjU7VxZ4Dtimeout", 500000


@dataclass
class TurnstileRequirementsProfile:
    screen_sum: int = 0
    user_agent: str = ""
    language: str = ""
    languages_join: str = ""
    navigator_probe: str = ""
    document_probe: str = ""
    window_probe: str = ""
    performance_now: float = 0.0
    session_id: str = ""
    hardware_concurrency: int = 0
    time_origin: float = 0.0


def parse_turnstile_requirements_profile(requirements_token: str) -> TurnstileRequirementsProfile:
    token = (requirements_token or "").strip()
    token = token.removeprefix("gAAAAAC").removeprefix("gAAAAAB").removesuffix("~S")
    if not token:
        raise ValueError("empty requirements token")
    body = base64.b64decode(token.encode("ascii"))
    fields = json.loads(body.decode("utf-8"))
    if not isinstance(fields, list) or len(fields) < 18:
        raise ValueError(f"invalid requirements field count: {len(fields) if isinstance(fields, list) else 'non-list'}")
    return TurnstileRequirementsProfile(
        screen_sum=int(json_float(fields[0])),
        user_agent=json_string(fields[4]),
        language=json_string(fields[7]),
        languages_join=json_string(fields[8]),
        navigator_probe=json_string(fields[10]),
        document_probe=json_string(fields[11]),
        window_probe=json_string(fields[12]),
        performance_now=json_float(fields[13]),
        session_id=json_string(fields[14]),
        hardware_concurrency=int(json_float(fields[16])),
        time_origin=json_float(fields[17]),
    )


class RegMapRef:
    def __init__(self, solver: "TurnstileSolver"):
        self.solver = solver


class SessionObserverMapRef:
    def __init__(self, solver: "SessionObserverSolver"):
        self.solver = solver


AuthWindowKeyOrder = [
    "0", "window", "self", "document", "name", "location", "customElements", "history", "navigation", "locationbar",
    "menubar", "personalbar", "scrollbars", "statusbar", "toolbar", "status", "closed", "frames", "length", "top",
    "opener", "parent", "frameElement", "navigator", "origin", "external", "screen", "innerWidth", "innerHeight",
    "scrollX", "pageXOffset", "scrollY", "pageYOffset", "visualViewport", "screenX", "screenY", "outerWidth",
    "outerHeight", "devicePixelRatio", "event", "clientInformation", "screenLeft", "screenTop", "styleMedia",
    "onsearch", "trustedTypes", "performance", "onappinstalled", "onbeforeinstallprompt", "crypto", "indexedDB",
    "sessionStorage", "localStorage", "onbeforexrselect", "onabort", "onbeforeinput", "onbeforematch",
    "onbeforetoggle", "onblur", "oncancel", "oncanplay", "oncanplaythrough", "onchange", "onclick", "onclose",
    "oncommand", "oncontentvisibilityautostatechange", "oncontextlost", "oncontextmenu", "oncontextrestored",
    "oncuechange", "ondblclick", "ondrag", "ondragend", "ondragenter", "ondragleave", "ondragover", "ondragstart",
    "ondrop", "ondurationchange", "onemptied", "onended", "onerror", "onfocus", "onformdata", "oninput", "oninvalid",
    "onkeydown", "onkeypress", "onkeyup", "onload", "onloadeddata", "onloadedmetadata", "onloadstart", "onmousedown",
    "onmouseenter", "onmouseleave", "onmousemove", "onmouseout", "onmouseover", "onmouseup", "onmousewheel",
    "onpause", "onplay", "onplaying", "onprogress", "onratechange", "onreset", "onresize", "onscroll", "onscrollend",
    "onsecuritypolicyviolation", "onseeked", "onseeking", "onselect", "onslotchange", "onstalled", "onsubmit",
    "onsuspend", "ontimeupdate", "ontoggle", "onvolumechange", "onwaiting", "onwebkitanimationend",
    "onwebkitanimationiteration", "onwebkitanimationstart", "onwebkittransitionend", "onwheel", "onauxclick",
    "ongotpointercapture", "onlostpointercapture", "onpointerdown", "onpointermove", "onpointerup", "onpointercancel",
    "onpointerover", "onpointerout", "onpointerenter", "onpointerleave", "onselectstart", "onselectionchange",
    "onanimationend", "onanimationiteration", "onanimationstart", "ontransitionrun", "ontransitionstart",
    "ontransitionend", "ontransitioncancel", "onafterprint", "onbeforeprint", "onbeforeunload", "onhashchange",
    "onlanguagechange", "onmessage", "onmessageerror", "onoffline", "ononline", "onpagehide", "onpageshow",
    "onpopstate", "onrejectionhandled", "onstorage", "onunhandledrejection", "onunload", "isSecureContext",
    "crossOriginIsolated", "scheduler", "alert", "atob", "blur", "btoa", "cancelAnimationFrame", "cancelIdleCallback",
    "captureEvents", "clearInterval", "clearTimeout", "close", "confirm", "createImageBitmap", "fetch", "find",
    "focus", "getComputedStyle", "getSelection", "matchMedia", "moveBy", "moveTo", "open", "postMessage", "print",
    "prompt", "queueMicrotask", "releaseEvents", "reportError", "requestAnimationFrame", "requestIdleCallback",
    "resizeBy", "resizeTo", "scroll", "scrollBy", "scrollTo", "setInterval", "setTimeout", "stop", "structuredClone",
    "webkitCancelAnimationFrame", "webkitRequestAnimationFrame", "chrome", "caches", "cookieStore", "ondevicemotion",
    "ondeviceorientation", "ondeviceorientationabsolute", "onpointerrawupdate", "documentPictureInPicture",
    "sharedStorage", "fetchLater", "getScreenDetails", "queryLocalFonts", "showDirectoryPicker", "showOpenFilePicker",
    "showSaveFilePicker", "originAgentCluster", "viewport", "onpageswap", "onpagereveal", "credentialless", "fence",
    "launchQueue", "speechSynthesis", "onscrollsnapchange", "onscrollsnapchanging", "webkitRequestFileSystem",
    "webkitResolveLocalFileSystemURL", "__reactRouterContext", "$RB", "$RV", "$RC", "$RT", "__reactRouterManifest",
    "__STATSIG__", "__reactRouterVersion", "__REACT_INTL_CONTEXT__", "DD_RUM", "__SEGMENT_INSPECTOR__",
    "__reactRouterRouteModules", "__reactRouterDataRouter", "__sentinel_token_pending", "__sentinel_init_pending",
    "SentinelSDK", "rwha4gh7no",
]

AuthNavigatorPrototypeKeys = [
    "vendorSub", "productSub", "vendor", "maxTouchPoints", "scheduling", "userActivation", "geolocation", "doNotTrack",
    "connection", "plugins", "mimeTypes", "pdfViewerEnabled", "webkitTemporaryStorage", "webkitPersistentStorage",
    "hardwareConcurrency", "cookieEnabled", "appCodeName", "appName", "appVersion", "platform", "product", "userAgent",
    "language", "languages", "onLine", "webdriver", "getGamepads", "javaEnabled", "sendBeacon", "vibrate",
    "windowControlsOverlay", "deprecatedRunAdAuctionEnforcesKAnonymity", "protectedAudience", "bluetooth",
    "storageBuckets", "clipboard", "credentials", "keyboard", "managed", "mediaDevices", "storage", "serviceWorker",
    "virtualKeyboard", "wakeLock", "deviceMemory", "userAgentData", "login", "ink", "mediaCapabilities",
    "devicePosture", "hid", "locks", "gpu", "mediaSession", "permissions", "presentation", "serial", "usb", "xr",
    "adAuctionComponents", "runAdAuction", "canLoadAdAuctionFencedFrame", "canShare", "share", "clearAppBadge",
    "getBattery", "getUserMedia", "requestMIDIAccess", "requestMediaKeySystemAccess", "setAppBadge", "webkitGetUserMedia",
    "clearOriginJoinedAdInterestGroups", "createAuctionNonce", "joinAdInterestGroup", "leaveAdInterestGroup",
    "updateAdInterestGroups", "deprecatedReplaceInURN", "deprecatedURNToURL", "getInstalledRelatedApps",
    "getInterestGroupAdAuctionData", "registerProtocolHandler", "unregisterProtocolHandler",
]


class TurnstileSolver:
    def __init__(self, requirements_token: str, dx: str, session: Session | None = None):
        self.session = session
        self.profile = parse_turnstile_requirements_profile(requirements_token)
        self.requirements_token = requirements_token
        self.dx = dx
        self.regs: dict[str, Any] = {}
        self.window: dict[str, Any] = {}
        self.done = False
        self.resolved = ""
        self.rejected = ""
        self.step_count = 0
        self.max_steps = 50000

    def solve(self) -> str:
        self.regs = {}
        self.window = self.build_window()
        self.done = False
        self.resolved = ""
        self.rejected = ""
        self.step_count = 0
        self.init_runtime()
        self.set_reg(TURNSTILE_SUCCESS_REG, self._success_fn)
        self.set_reg(TURNSTILE_ERROR_REG, self._error_fn)
        self.set_reg(TURNSTILE_CALLBACK_REG, self._callback_reg_fn)
        self.set_reg(TURNSTILE_KEY_REG, self.requirements_token)

        decoded = latin1_base64_decode(self.dx)
        plain = xor_string(decoded, self.requirements_token)
        queue = json.loads(plain)
        self.set_reg(TURNSTILE_QUEUE_REG, queue)
        try:
            self.run_queue()
        except Exception as exc:
            if not self.done:
                self._success_fn(f"{self.step_count}: {exc}")
        if self.rejected:
            raise RuntimeError(self.rejected)
        if not self.resolved:
            raise RuntimeError(f"turnstile vm unresolved after {self.step_count} steps")
        return self.resolved

    def _success_fn(self, *args: Any) -> None:
        if not self.done:
            self.done = True
            value = args[0] if args else None
            self.resolved = latin1_base64_encode(self.js_to_string(value))

    def _error_fn(self, *args: Any) -> None:
        if not self.done:
            self.done = True
            value = args[0] if args else None
            self.rejected = latin1_base64_encode(self.js_to_string(value))

    def _callback_reg_fn(self, *args: Any) -> None:
        if len(args) < 3:
            return None
        target_reg, return_reg = args[0], args[1]
        arg_regs = args[2] if isinstance(args[2], list) else []
        inner_queue = args[3] if len(args) >= 4 and isinstance(args[3], list) else arg_regs
        mapped_arg_regs = arg_regs if len(args) >= 4 else []

        def callback(*call_args: Any) -> Any:
            if self.done:
                return None
            previous_queue = self.copy_queue()
            for index, reg_id in enumerate(mapped_arg_regs):
                self.set_reg(reg_id, call_args[index] if index < len(call_args) else None)
            self.set_reg(TURNSTILE_QUEUE_REG, copy_any_slice(inner_queue))
            err = None
            try:
                self.run_queue()
            except Exception as exc:
                err = exc
            self.set_reg(TURNSTILE_QUEUE_REG, previous_queue)
            if err is not None:
                return str(err)
            return self.get_reg(return_reg)

        self.set_reg(target_reg, callback)
        return None

    def init_runtime(self) -> None:
        self.set_reg(0, self._op_solve_nested)
        self.set_reg(1, self._op_xor)
        self.set_reg(2, self._op_assign_literal)
        self.set_reg(5, self._op_add)
        self.set_reg(6, self._op_get_prop)
        self.set_reg(7, self._op_call_void)
        self.set_reg(8, self._op_assign_reg)
        self.set_reg(11, self._op_match_script_src)
        self.set_reg(12, self._op_make_reg_map_ref)
        self.set_reg(13, self._op_call_capture_error)
        self.set_reg(14, self._op_json_parse)
        self.set_reg(15, self._op_json_stringify)
        self.set_reg(17, self._op_call_assign)
        self.set_reg(18, self._op_atob)
        self.set_reg(19, self._op_btoa)
        self.set_reg(20, self._op_if_equal)
        self.set_reg(21, self._op_if_abs_gt)
        self.set_reg(22, self._op_run_subqueue)
        self.set_reg(23, self._op_if_defined)
        self.set_reg(24, self._op_bind_method)
        self.set_reg(25, lambda *args: None)
        self.set_reg(26, lambda *args: None)
        self.set_reg(27, self._op_sub_or_remove)
        self.set_reg(28, lambda *args: None)
        self.set_reg(29, self._op_lt)
        self.set_reg(33, self._op_mul)
        self.set_reg(34, self._op_assign_reg)
        self.set_reg(35, self._op_div)
        self.set_reg(TURNSTILE_WINDOW_REG, self.window)

    def _op_solve_nested(self, *args: Any) -> Any:
        if not args:
            return None
        return solve_turnstile_dx_with_session(self.js_to_string(self.get_reg(TURNSTILE_KEY_REG)), self.js_to_string(args[0]), self.session)

    def _op_xor(self, *args: Any) -> None:
        if len(args) >= 2:
            target, key_reg = args[0], args[1]
            self.set_reg(target, xor_string(self.js_to_string(self.get_reg(target)), self.js_to_string(self.get_reg(key_reg))))

    def _op_assign_literal(self, *args: Any) -> None:
        if len(args) >= 2:
            self.set_reg(args[0], args[1])

    def _op_add(self, *args: Any) -> None:
        if len(args) < 2:
            return
        left = self.get_reg(args[0])
        right = self.get_reg(args[1])
        if isinstance(left, list):
            self.set_reg(args[0], [*left, right])
            return
        left_num, left_ok = self.as_number(left)
        right_num, right_ok = self.as_number(right)
        if left_ok and right_ok:
            self.set_reg(args[0], left_num + right_num)
            return
        self.set_reg(args[0], self.js_to_string(left) + self.js_to_string(right))

    def _op_get_prop(self, *args: Any) -> None:
        if len(args) >= 3:
            self.set_reg(args[0], self.js_get_prop(self.get_reg(args[1]), self.get_reg(args[2])))

    def _op_call_void(self, *args: Any) -> None:
        if args:
            self.call_fn(self.get_reg(args[0]), *self.deref_args(list(args[1:])))

    def _op_assign_reg(self, *args: Any) -> None:
        if len(args) >= 2:
            self.set_reg(args[0], self.get_reg(args[1]))

    def _op_match_script_src(self, *args: Any) -> None:
        if len(args) < 2:
            return
        pattern = self.js_to_string(self.get_reg(args[1]))
        try:
            rx = re.compile(pattern)
        except re.error:
            self.set_reg(args[0], None)
            return
        scripts = self.js_get_prop(self.js_get_prop(self.window, "document"), "scripts") or []
        for item in scripts:
            src = self.js_to_string(self.js_get_prop(item, "src"))
            if src and rx.search(src):
                self.set_reg(args[0], rx.search(src).group(0))
                return
        self.set_reg(args[0], None)

    def _op_make_reg_map_ref(self, *args: Any) -> None:
        if args:
            self.set_reg(args[0], RegMapRef(self))

    def _op_call_capture_error(self, *args: Any) -> None:
        if len(args) < 2:
            return
        try:
            self.call_fn(self.get_reg(args[1]), *list(args[2:]))
        except Exception as exc:
            self.set_reg(args[0], str(exc))

    def _op_json_parse(self, *args: Any) -> None:
        if len(args) < 2:
            return
        self.set_reg(args[0], json.loads(self.js_to_string(self.get_reg(args[1]))))

    def _op_json_stringify(self, *args: Any) -> None:
        if len(args) < 2:
            return
        self.set_reg(args[0], js_json_stringify(self.get_reg(args[1])))

    def _op_call_assign(self, *args: Any) -> None:
        if len(args) < 2:
            return
        try:
            value = self.call_fn(self.get_reg(args[1]), *self.deref_args(list(args[2:])))
        except Exception as exc:
            self.set_reg(args[0], str(exc))
            return
        self.set_reg(args[0], value)

    def _op_atob(self, *args: Any) -> None:
        if args:
            self.set_reg(args[0], latin1_base64_decode(self.js_to_string(self.get_reg(args[0]))))

    def _op_btoa(self, *args: Any) -> None:
        if args:
            self.set_reg(args[0], latin1_base64_encode(self.js_to_string(self.get_reg(args[0]))))

    def _op_if_equal(self, *args: Any) -> None:
        if len(args) >= 3 and self.values_equal(self.get_reg(args[0]), self.get_reg(args[1])):
            self.call_fn(self.get_reg(args[2]), *list(args[3:]))

    def _op_if_abs_gt(self, *args: Any) -> None:
        if len(args) < 4:
            return
        left, _ = self.as_number(self.get_reg(args[0]))
        right, _ = self.as_number(self.get_reg(args[1]))
        threshold, _ = self.as_number(self.get_reg(args[2]))
        if abs(left - right) > threshold:
            self.call_fn(self.get_reg(args[3]), *list(args[4:]))

    def _op_run_subqueue(self, *args: Any) -> None:
        if len(args) < 2:
            return
        previous_queue = self.copy_queue()
        self.set_reg(TURNSTILE_QUEUE_REG, copy_any_slice(args[1] if isinstance(args[1], list) else []))
        try:
            self.run_queue()
        except Exception as exc:
            self.set_reg(args[0], str(exc))
        finally:
            self.set_reg(TURNSTILE_QUEUE_REG, previous_queue)

    def _op_if_defined(self, *args: Any) -> None:
        if len(args) >= 2 and self.get_reg(args[0]) is not None:
            self.call_fn(self.get_reg(args[1]), *list(args[2:]))

    def _op_bind_method(self, *args: Any) -> None:
        if len(args) < 3:
            return
        method = self.js_get_prop(self.get_reg(args[1]), self.get_reg(args[2]))
        self.set_reg(args[0], method if callable(method) else None)

    def _op_sub_or_remove(self, *args: Any) -> None:
        if len(args) < 2:
            return
        left = self.get_reg(args[0])
        right = self.get_reg(args[1])
        if isinstance(left, list):
            self.set_reg(args[0], [item for item in left if not self.values_equal(item, right)])
            return
        left_num, left_ok = self.as_number(left)
        right_num, right_ok = self.as_number(right)
        if left_ok and right_ok:
            self.set_reg(args[0], left_num - right_num)

    def _op_lt(self, *args: Any) -> None:
        if len(args) >= 3:
            left, _ = self.as_number(self.get_reg(args[1]))
            right, _ = self.as_number(self.get_reg(args[2]))
            self.set_reg(args[0], left < right)

    def _op_mul(self, *args: Any) -> None:
        if len(args) >= 3:
            left, _ = self.as_number(self.get_reg(args[1]))
            right, _ = self.as_number(self.get_reg(args[2]))
            self.set_reg(args[0], left * right)

    def _op_div(self, *args: Any) -> None:
        if len(args) >= 3:
            left, _ = self.as_number(self.get_reg(args[1]))
            right, _ = self.as_number(self.get_reg(args[2]))
            self.set_reg(args[0], 0.0 if right == 0 else left / right)

    def build_window(self) -> dict[str, Any]:
        ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36"
        lang = "zh-CN"
        languages_join = "zh-CN,en-US"
        width = 2048
        height = 1152
        inner_width = 800
        inner_height = 600
        outer_width = 160
        outer_height = 28
        screen_x = -25600
        screen_y = -25600
        hardware_concurrency = 8
        heap_limit = 4294705152
        device_id = "bb13486d-db99-4547-81a4-a8f2a6351be9"
        time_origin = float(int((time.time() - 10) * 1000))
        performance_now = 9270.399999976158
        vendor = "Google Inc."
        platform = "Win32"
        document_probe = "__reactContainer$b63yiita51i"
        window_probe = "ondragend"

        if self.profile.screen_sum > 0:
            width, height = split_screen_sum(self.profile.screen_sum)
        if self.profile.user_agent.strip():
            ua = self.profile.user_agent
        if self.profile.language.strip():
            lang = self.profile.language
        if self.profile.languages_join.strip():
            languages_join = self.profile.languages_join
        if self.profile.hardware_concurrency > 0:
            hardware_concurrency = self.profile.hardware_concurrency
        if self.profile.time_origin > 0:
            time_origin = self.profile.time_origin
        if self.profile.performance_now > 0:
            performance_now = self.profile.performance_now
        if self.profile.document_probe.strip():
            document_probe = self.profile.document_probe.strip()
        if self.profile.window_probe.strip():
            window_probe = self.profile.window_probe.strip()
        if self.session:
            persona = self.session.persona
            if self.session.user_agent.strip():
                ua = self.session.user_agent
            if self.session.language.strip():
                lang = self.session.language
            if self.session.languages_join.strip():
                languages_join = self.session.languages_join
            if self.session.screen_width > 0:
                width = self.session.screen_width
            if self.session.screen_height > 0:
                height = self.session.screen_height
            if self.session.hardware_concurrency > 0:
                hardware_concurrency = self.session.hardware_concurrency
            if self.session.heap_limit > 0:
                heap_limit = self.session.heap_limit
            if self.session.device_id.strip():
                device_id = self.session.device_id
            if persona.time_origin > 0:
                time_origin = persona.time_origin
            if persona.vendor.strip():
                vendor = persona.vendor
            if persona.platform.strip():
                platform = persona.platform
            if persona.performance_now > 0 and self.profile.performance_now <= 0:
                performance_now = persona.performance_now
            if persona.document_probe.strip() and not self.profile.document_probe.strip():
                document_probe = persona.document_probe.strip()
            if persona.window_probe.strip() and not self.profile.window_probe.strip():
                window_probe = persona.window_probe.strip()

        location = with_ordered_keys({"href": "https://auth.openai.com/create-account/password", "search": ""}, [])
        scripts = [
            with_ordered_keys({"src": "https://sentinel.openai.com/backend-api/sentinel/sdk.js"}, []),
            with_ordered_keys({"src": "https://sentinel.openai.com/sentinel/20260219f9f6/sdk.js"}, []),
        ]
        local_storage_data = with_ordered_keys({
            "statsig.stable_id.444584300": f'"{device_id}"',
            "statsig.session_id.444584300": '{"sessionID":"acba2013-7acb-405b-8064-7917fe2d8b4d","startTime":1775195126921,"lastUpdate":1775195156096}',
            "statsig.network_fallback.2742193661": '{"initialize":{"urlConfigChecksum":"3392903","url":"https://assetsconfigcdn.org/v1/initialize","expiryTime":1775799927542,"previous":[]}}',
        }, [
            "statsig.stable_id.444584300",
            "statsig.session_id.444584300",
            "statsig.network_fallback.2742193661",
        ])
        storage_keys = keys_of_map(local_storage_data)
        local_storage = with_ordered_keys({
            "__storage_data__": local_storage_data,
            "__storage_keys__": list(storage_keys),
            "length": float(len(storage_keys)),
        }, [])

        def refresh_local_storage_meta() -> None:
            new_keys = keys_of_map(local_storage_data)
            local_storage["__storage_keys__"] = list(new_keys)
            local_storage["length"] = float(len(new_keys))

        local_storage["key"] = lambda *args: list(storage_keys)[to_int_index(args[0])] if args and 0 <= to_int_index(args[0]) < len(storage_keys) else None
        local_storage["getItem"] = lambda *args: local_storage_data.get(self.js_to_string(args[0])) if args else None
        def set_item(*args: Any) -> None:
            if len(args) < 2:
                return
            local_storage_data[self.js_to_string(args[0])] = self.js_to_string(args[1])
            refresh_local_storage_meta()
        local_storage["setItem"] = set_item
        def remove_item(*args: Any) -> None:
            if args:
                local_storage_data.pop(self.js_to_string(args[0]), None)
                refresh_local_storage_meta()
        local_storage["removeItem"] = remove_item
        def clear_item(*args: Any) -> None:
            local_storage_data.clear()
            refresh_local_storage_meta()
        local_storage["clear"] = clear_item

        screen = with_ordered_keys({
            "availWidth": float(width),
            "availHeight": float(height),
            "availLeft": 0.0,
            "availTop": 0.0,
            "colorDepth": 24.0,
            "pixelDepth": 24.0,
            "width": float(width),
            "height": float(height),
        }, [])
        document = with_ordered_keys({
            "scripts": scripts,
            "location": location,
            "documentElement": with_ordered_keys({"getAttribute": lambda *args: None}, []),
        }, ["location", document_probe, "_reactListeningj3rmi50kcy", "closure_lm_184788"])
        document["body"] = with_ordered_keys({
            "getBoundingClientRect": lambda *args: with_ordered_keys({
                "x": 0.0, "y": 0.0, "width": 800.0, "height": 346.0, "top": 0.0, "left": 0.0, "right": 800.0, "bottom": 346.0,
            }, [])
        }, [])
        document["getElementById"] = lambda *args: document["body"]
        document["querySelector"] = lambda *args: document["body"]

        def create_element(*args: Any) -> dict[str, Any]:
            tag = (self.js_to_string(args[0]) if args else "").lower()
            element = with_ordered_keys({
                "tagName": tag.upper(),
                "style": with_ordered_keys({}, []),
                "appendChild": lambda *a: a[0] if a else None,
                "removeChild": lambda *a: a[0] if a else None,
                "remove": lambda *a: None,
            }, [])
            if tag == "canvas":
                element["getContext"] = lambda *a: with_ordered_keys({
                    "getExtension": lambda *b: with_ordered_keys({"UNMASKED_VENDOR_WEBGL": 37445.0, "UNMASKED_RENDERER_WEBGL": 37446.0}, []) if b and self.js_to_string(b[0]) == "WEBGL_debug_renderer_info" else None,
                    "getParameter": lambda *b: "Google Inc. (NVIDIA)" if b and to_int_index(b[0]) in {37445, 7936} else ("ANGLE (NVIDIA, NVIDIA GeForce RTX 5080 Laptop GPU Direct3D11 vs_5_0 ps_5_0, D3D11)" if b and to_int_index(b[0]) in {37446, 7937} else None),
                }, [])
            return element
        document["createElement"] = create_element

        navigator = with_ordered_keys({
            "userAgent": ua,
            "vendor": vendor,
            "platform": platform,
            "hardwareConcurrency": float(hardware_concurrency),
            "deviceMemory": 8.0,
            "maxTouchPoints": 10.0,
            "language": lang,
            "languages": string_slice_to_any([item.strip() for item in languages_join.replace(";q=0.9", "").split(",")]),
            "webdriver": False,
        }, [])
        navigator["clipboard"] = with_ordered_keys({}, [])
        navigator["xr"] = with_ordered_keys({}, [])
        navigator["storage"] = with_ordered_keys({
            "estimate": lambda *args: with_ordered_keys({"quota": 306461727129.0, "usage": 0.0, "usageDetails": with_ordered_keys({}, [])}, [])
        }, [])
        navigator["userAgentData"] = with_ordered_keys({
            "brands": [
                with_ordered_keys({"brand": "Chromium", "version": "142"}, []),
                with_ordered_keys({"brand": "Google Chrome", "version": "142"}, []),
                with_ordered_keys({"brand": "Not_A Brand", "version": "99"}, []),
            ],
            "mobile": False,
            "platform": "Windows",
            "getHighEntropyValues": lambda *args: with_ordered_keys({
                "platform": "Windows",
                "platformVersion": "10.0.0",
                "architecture": "x86",
                "model": "",
                "uaFullVersion": "142.0.0.0",
            }, []),
        }, [])
        navigator[PROTOTYPE_META] = with_ordered_keys({}, AuthNavigatorPrototypeKeys)
        for key in AuthNavigatorPrototypeKeys:
            navigator[PROTOTYPE_META].setdefault(key, None)

        start = time.monotonic()
        window = with_ordered_keys({}, AuthWindowKeyOrder)
        window["Reflect"] = with_ordered_keys({"set": lambda *args: self.js_set_prop(args[0], args[1], args[2]) if len(args) >= 3 else True}, [])
        window["Object"] = with_ordered_keys({
            "keys": lambda *args: object_keys(args[0]) if args else [],
            "getPrototypeOf": lambda *args: args[0].get(PROTOTYPE_META) if args and isinstance(args[0], dict) else None,
            "create": lambda *args: with_ordered_keys({}, []),
        }, [])
        window["Math"] = with_ordered_keys({
            "random": lambda *args: _SEEDED.random(),
            "abs": lambda *args: abs(self.as_number(args[0])[0]) if args else 0.0,
        }, [])
        window["JSON"] = with_ordered_keys({
            "parse": lambda *args: json.loads(self.js_to_string(args[0])) if args else None,
            "stringify": lambda *args: js_json_stringify(args[0]) if args else "null",
        }, [])
        window["atob"] = lambda *args: latin1_base64_decode(self.js_to_string(args[0])) if args else ""
        window["btoa"] = lambda *args: latin1_base64_encode(self.js_to_string(args[0])) if args else ""
        window["localStorage"] = local_storage
        window["document"] = document
        window["navigator"] = navigator
        window["screen"] = screen
        window["location"] = location
        window["history"] = with_ordered_keys({"length": 3.0}, [])
        window["performance"] = with_ordered_keys({
            "now": lambda *args: performance_now + (time.monotonic() - start) * 1000.0,
            "timeOrigin": time_origin,
            "memory": with_ordered_keys({"jsHeapSizeLimit": float(heap_limit)}, []),
        }, [])
        window["Array"] = with_ordered_keys({
            "from": lambda *args: list(args[0]) if args and isinstance(args[0], (list, tuple)) else [],
        }, [])
        window.update({
            "0": None,
            "innerWidth": float(inner_width),
            "innerHeight": float(inner_height),
            "outerWidth": float(outer_width),
            "outerHeight": float(outer_height),
            "screenX": float(screen_x),
            "screenY": float(screen_y),
            "scrollX": 0.0,
            "pageXOffset": 0.0,
            "scrollY": 0.0,
            "pageYOffset": 0.0,
            "devicePixelRatio": 1.0000000149011612,
            "hardwareConcurrency": float(hardware_concurrency),
            "isSecureContext": True,
            "crossOriginIsolated": False,
            "event": None,
            "clientInformation": navigator,
            "screenLeft": float(screen_x),
            "screenTop": float(screen_y),
            "chrome": with_ordered_keys({"runtime": with_ordered_keys({}, [])}, []),
            "__reactRouterContext": with_ordered_keys({
                "future": with_ordered_keys({}, []),
                "routeDiscovery": with_ordered_keys({}, []),
                "ssr": True,
                "isSpaMode": True,
                "loaderData": with_ordered_keys({
                    "routes/layouts/client-auth-session-layout/layout": with_ordered_keys({
                        "session": with_ordered_keys({
                            "session_id": self.profile.session_id,
                            "auth_session_logging_id": "3691f3d7-3e89-440c-99c1-0788585b7688",
                        }, [])
                    }, [])
                }, []),
            }, []),
            "$RB": [],
            "$RV": lambda *args: None,
            "$RC": lambda *args: None,
            "$RT": performance_now,
            "__reactRouterManifest": with_ordered_keys({}, []),
            "__STATSIG__": with_ordered_keys({}, []),
            "__reactRouterVersion": "7.9.3",
            "__REACT_INTL_CONTEXT__": with_ordered_keys({}, []),
            "DD_RUM": with_ordered_keys({}, []),
            "__SEGMENT_INSPECTOR__": with_ordered_keys({}, []),
            "__reactRouterRouteModules": with_ordered_keys({}, []),
            "__reactRouterDataRouter": with_ordered_keys({}, []),
            "__sentinel_token_pending": with_ordered_keys({}, []),
            "__sentinel_init_pending": with_ordered_keys({}, []),
            "SentinelSDK": with_ordered_keys({}, []),
            "rwha4gh7no": lambda *args: None,
        })
        window[window_probe] = None
        append_ordered_key(window, window_probe)
        document[document_probe] = None
        document["_reactListeningj3rmi50kcy"] = True
        document["closure_lm_184788"] = None
        window["window"] = window
        window["self"] = window
        window["globalThis"] = window
        for key in AuthWindowKeyOrder:
            window.setdefault(key, None)
        window["0"] = window
        return window

    def run_queue(self) -> None:
        while not self.done:
            queue = self.get_reg(TURNSTILE_QUEUE_REG)
            if not isinstance(queue, list) or not queue:
                return
            ins = queue[0]
            self.set_reg(TURNSTILE_QUEUE_REG, queue[1:])
            if not isinstance(ins, list) or not ins:
                continue
            fn = self.get_reg(ins[0])
            if not callable(fn):
                raise RuntimeError(f"vm opcode not callable: {ins[0]}")
            fn(*ins[1:])
            self.step_count += 1
            if self.step_count > self.max_steps:
                raise RuntimeError("turnstile vm step overflow")

    def call_fn(self, value: Any, *args: Any) -> Any:
        if not callable(value):
            return None
        return value(*args)

    def deref_args(self, args: list[Any]) -> list[Any]:
        return [self.get_reg(arg) for arg in args]

    def get_reg(self, key: Any) -> Any:
        return self.regs.get(reg_key(key))

    def set_reg(self, key: Any, value: Any) -> None:
        self.regs[reg_key(key)] = value

    def copy_queue(self) -> list[Any]:
        queue = self.get_reg(TURNSTILE_QUEUE_REG)
        return copy_any_slice(queue if isinstance(queue, list) else [])

    def as_number(self, value: Any) -> tuple[float, bool]:
        if value is None:
            return math.nan, False
        if isinstance(value, bool):
            return (1.0 if value else 0.0), True
        if isinstance(value, (int, float)):
            return float(value), True
        if isinstance(value, str):
            if not value.strip():
                return 0.0, True
            try:
                return float(value.strip()), True
            except ValueError:
                return math.nan, False
        return math.nan, False

    def values_equal(self, left: Any, right: Any) -> bool:
        if isinstance(left, float) and isinstance(right, float):
            return left == right
        if isinstance(left, (str, bool)) and isinstance(right, type(left)):
            return left == right
        if left is None:
            return right is None
        return f"{left}" == f"{right}"

    def js_to_string(self, value: Any) -> str:
        if value is None:
            return "undefined"
        if isinstance(value, bool):
            return "true" if value else "false"
        if isinstance(value, int):
            return str(value)
        if isinstance(value, float):
            if math.isnan(value):
                return "NaN"
            if math.isinf(value):
                return "Infinity" if value > 0 else "-Infinity"
            if math.trunc(value) == value:
                return str(int(value))
            return format(value, "f").rstrip("0").rstrip(".")
        if isinstance(value, str):
            return value
        if isinstance(value, list):
            return ",".join(self.js_to_string_array_item(item) for item in value)
        if isinstance(value, dict):
            if isinstance(value.get("href"), str) and value.get("search") is not None:
                return value["href"]
            return "[object Object]"
        return str(value)

    def js_to_string_array_item(self, value: Any) -> str:
        if value is None:
            return ""
        return self.js_to_string(value)

    def js_get_prop(self, obj: Any, prop: Any) -> Any:
        if obj is None:
            return None
        if isinstance(obj, RegMapRef):
            return obj.solver.get_reg(prop)
        if isinstance(obj, dict):
            prop_key = self.js_to_string(prop)
            if "__storage_data__" in obj and isinstance(obj["__storage_data__"], dict):
                storage = obj["__storage_data__"]
                if prop_key in {"__storage_data__", "__storage_keys__", "length", "key", "getItem", "setItem", "removeItem", "clear"}:
                    return obj.get(prop_key)
                return obj.get(prop_key, storage.get(prop_key))
            return obj.get(prop_key)
        if isinstance(obj, list):
            if self.js_to_string(prop) == "length":
                return float(len(obj))
            index = to_int_index(prop)
            return obj[index] if 0 <= index < len(obj) else None
        if isinstance(obj, str):
            if self.js_to_string(prop) == "length":
                return float(len(obj))
            index = to_int_index(prop)
            return obj[index] if 0 <= index < len(obj) else None
        return None

    def js_set_prop(self, obj: Any, prop: Any, value: Any) -> bool:
        if isinstance(obj, RegMapRef):
            obj.solver.set_reg(prop, value)
            return True
        if isinstance(obj, dict):
            prop_key = self.js_to_string(prop)
            if "__storage_data__" in obj and isinstance(obj["__storage_data__"], dict):
                storage = obj["__storage_data__"]
                if prop_key in {"__storage_data__", "__storage_keys__", "length", "key", "getItem", "setItem", "removeItem", "clear"}:
                    obj[prop_key] = value
                else:
                    storage[prop_key] = value
                    obj[prop_key] = value
                    obj["__storage_keys__"] = keys_of_map(storage)
                    obj["length"] = float(len(keys_of_map(storage)))
                return True
            obj[prop_key] = value
            append_ordered_key(obj, prop_key)
            return True
        return False


class SessionObserverSolver(TurnstileSolver):
    def __init__(self, proof: str, collector_dx: str, session: Session | None = None):
        self.session = session
        self.profile = parse_turnstile_requirements_profile(proof)
        self.proof = proof
        self.collector_dx = collector_dx
        self.regs: dict[str, Any] = {}
        self.window: dict[str, Any] = {}
        self.done = False
        self.resolved = ""
        self.rejected = ""
        self.step_count = 0
        self.max_steps = 60000

    def solve(self) -> str:
        self.regs = {}
        self.window = self.build_window()
        self.done = False
        self.resolved = ""
        self.rejected = ""
        self.step_count = 0
        self._init_runtime()
        self.set_reg(TURNSTILE_KEY_REG, self.proof)
        self.set_reg(TURNSTILE_SUCCESS_REG, self._success_cb)
        self.set_reg(TURNSTILE_ERROR_REG, self._error_cb)
        self.set_reg(30, self._callback_builder)

        decoded = latin1_base64_decode(self.collector_dx)
        queue = json.loads(xor_string(decoded, self.js_to_string(self.get_reg(TURNSTILE_KEY_REG))))
        self.set_reg(TURNSTILE_QUEUE_REG, queue)
        try:
            self.run_queue()
            if not self.done:
                self._finish_success(latin1_base64_encode(f"{self.step_count}: {self.js_to_string(self.get_reg(TURNSTILE_SUCCESS_REG))}"))
        except Exception as exc:
            if not self.done:
                self._finish_success(latin1_base64_encode(f"{self.step_count}: {exc}"))
        if self.rejected:
            raise RuntimeError(self.rejected)
        if not self.resolved:
            raise RuntimeError("session observer unresolved")
        return self.resolved

    def _finish_success(self, value: str) -> None:
        if not self.done:
            self.done = True
            self.resolved = value

    def _finish_error(self, value: str) -> None:
        if not self.done:
            self.done = True
            self.rejected = value

    def _success_cb(self, *args: Any) -> None:
        value = args[0] if args else ""
        self._finish_success(latin1_base64_encode(f"{value}"))

    def _error_cb(self, *args: Any) -> None:
        value = args[0] if args else ""
        self._finish_error(latin1_base64_encode(f"{value}"))

    def _callback_builder(self, *args: Any) -> None:
        if len(args) < 4:
            return None
        target_reg, next_reg, arg_regs, queue_or_regs = args[0], args[1], args[2], args[3]
        use_arg_mapping = isinstance(queue_or_regs, list)
        mapped_arg_regs = arg_regs if use_arg_mapping and isinstance(arg_regs, list) else []
        callback_queue = queue_or_regs if use_arg_mapping else arg_regs
        callback_queue = callback_queue if isinstance(callback_queue, list) else []

        def callback(*cb_args: Any) -> Any:
            previous_queue = self.copy_queue()
            previous_snapshot = dict(self.regs)
            if use_arg_mapping:
                for idx, reg_id in enumerate(mapped_arg_regs):
                    self.set_reg(reg_id, cb_args[idx] if idx < len(cb_args) else None)
            self.set_reg(TURNSTILE_QUEUE_REG, copy_any_slice(callback_queue))
            try:
                self.run_queue()
                return self.get_reg(next_reg)
            except Exception as exc:
                return str(exc)
            finally:
                self.regs = previous_snapshot
                self.set_reg(TURNSTILE_QUEUE_REG, previous_queue)

        self.set_reg(target_reg, callback)
        return None

    def _init_runtime(self) -> None:
        self.set_reg(0, lambda *args: solve_session_observer_with_proof(self.js_to_string(args[0]) if args else "", self.js_to_string(self.get_reg(TURNSTILE_KEY_REG)), self.session))
        self.set_reg(1, lambda target, key_reg: self.set_reg(target, xor_string(str(self.get_reg(target)), str(self.get_reg(key_reg)))))
        self.set_reg(2, lambda target, value: self.set_reg(target, value))
        self.set_reg(5, self._so_add)
        self.set_reg(27, self._so_sub_or_remove)
        self.set_reg(29, lambda target, left, right: self.set_reg(target, self.get_reg(left) < self.get_reg(right)))
        self.set_reg(33, lambda target, left, right: self.set_reg(target, float(self.get_reg(left)) * float(self.get_reg(right))))
        self.set_reg(35, self._so_div)
        self.set_reg(6, lambda target, obj_reg, prop_reg: self.set_reg(target, self.js_get_prop(self.get_reg(obj_reg), self.get_reg(prop_reg))))
        self.set_reg(7, lambda target, *regs: self.call_fn(self.get_reg(target), *[self.get_reg(reg) for reg in regs]))
        self.set_reg(8, lambda target, reg: self.set_reg(target, self.get_reg(reg)))
        self.set_reg(10, self.window)
        self.set_reg(11, self._so_match_script)
        self.set_reg(12, lambda target: self.set_reg(target, SessionObserverMapRef(self)))
        self.set_reg(14, lambda target, reg: self.set_reg(target, json.loads("" + self.js_to_string(self.get_reg(reg)))))
        self.set_reg(15, lambda target, reg: self.set_reg(target, json.dumps(self.get_reg(reg), ensure_ascii=False, separators=(",", ":"))))
        self.set_reg(17, self._so_call_assign)
        self.set_reg(13, self._so_call_capture_error)
        self.set_reg(18, lambda reg: self.set_reg(reg, latin1_base64_decode("" + self.js_to_string(self.get_reg(reg)))))
        self.set_reg(19, lambda reg: self.set_reg(reg, latin1_base64_encode("" + self.js_to_string(self.get_reg(reg)))))
        self.set_reg(20, self._so_if_equal)
        self.set_reg(21, self._so_if_abs_gt)
        self.set_reg(23, self._so_if_defined)
        self.set_reg(24, self._so_bind_method)
        self.set_reg(22, self._so_run_subqueue)
        self.set_reg(28, lambda *args: None)
        self.set_reg(25, lambda *args: None)
        self.set_reg(26, lambda *args: None)

    def _so_add(self, target: Any, reg: Any) -> None:
        left = self.get_reg(target)
        right = self.get_reg(reg)
        if isinstance(left, list):
            left.append(right)
            self.set_reg(target, left)
            return
        try:
            self.set_reg(target, left + right)
        except Exception:
            self.set_reg(target, f"{left}{right}")

    def _so_sub_or_remove(self, target: Any, reg: Any) -> None:
        left = self.get_reg(target)
        right = self.get_reg(reg)
        if isinstance(left, list):
            try:
                left.remove(right)
            except ValueError:
                pass
            self.set_reg(target, left)
            return
        try:
            self.set_reg(target, left - right)
        except Exception:
            pass

    def _so_div(self, target: Any, left_reg: Any, right_reg: Any) -> None:
        try:
            left = float(self.get_reg(left_reg))
            right = float(self.get_reg(right_reg))
        except Exception:
            self.set_reg(target, 0)
            return
        self.set_reg(target, 0 if right == 0 else left / right)

    def _so_match_script(self, target: Any, pattern_reg: Any) -> None:
        pattern = self.js_to_string(self.get_reg(pattern_reg))
        try:
            rx = re.compile(pattern)
        except re.error:
            self.set_reg(target, None)
            return
        scripts = self.js_get_prop(self.js_get_prop(self.window, "document"), "scripts") or []
        found = None
        for item in scripts:
            src = self.js_to_string(self.js_get_prop(item, "src"))
            if src and rx.search(src):
                found = rx.search(src).group(0)
                break
        self.set_reg(target, found)

    def _so_call_assign(self, target: Any, fn_reg: Any, *regs: Any) -> None:
        try:
            result = self.call_fn(self.get_reg(fn_reg), *[self.get_reg(reg) for reg in regs])
        except Exception as exc:
            self.set_reg(target, str(exc))
            return
        if hasattr(result, "__await__"):
            raise RuntimeError("async result not supported in pure python so-vm")
        self.set_reg(target, result)

    def _so_call_capture_error(self, target: Any, fn_reg: Any, *regs: Any) -> None:
        try:
            self.call_fn(self.get_reg(fn_reg), *regs)
        except Exception as exc:
            self.set_reg(target, str(exc))

    def _so_if_equal(self, left_reg: Any, right_reg: Any, fn_reg: Any, *regs: Any) -> None:
        if self.get_reg(left_reg) == self.get_reg(right_reg):
            self.call_fn(self.get_reg(fn_reg), *regs)

    def _so_if_abs_gt(self, left_reg: Any, right_reg: Any, threshold_reg: Any, fn_reg: Any, *regs: Any) -> None:
        try:
            left = float(self.get_reg(left_reg))
            right = float(self.get_reg(right_reg))
            threshold = float(self.get_reg(threshold_reg))
        except Exception:
            return
        if abs(left - right) > threshold:
            self.call_fn(self.get_reg(fn_reg), *regs)

    def _so_if_defined(self, reg: Any, fn_reg: Any, *rest: Any) -> None:
        if self.get_reg(reg) is not None:
            self.call_fn(self.get_reg(fn_reg), *rest)

    def _so_bind_method(self, target: Any, obj_reg: Any, prop_reg: Any) -> None:
        obj = self.get_reg(obj_reg)
        prop = self.get_reg(prop_reg)
        method = self.js_get_prop(obj, prop)
        self.set_reg(target, method if callable(method) else None)

    def _so_run_subqueue(self, target: Any, queue_reg: Any) -> None:
        previous_queue = self.copy_queue()
        queue = list(queue_reg) if isinstance(queue_reg, list) else []
        self.set_reg(TURNSTILE_QUEUE_REG, queue)
        try:
            self.run_queue()
        except Exception as exc:
            self.set_reg(target, str(exc))
        finally:
            self.set_reg(TURNSTILE_QUEUE_REG, previous_queue)

    def js_get_prop(self, obj: Any, prop: Any) -> Any:
        if isinstance(obj, SessionObserverMapRef):
            prop_key = self.js_to_string(prop)
            if prop_key == "set":
                return lambda key, value: obj.solver.set_reg(key, value)
            if prop_key == "get":
                return lambda key: obj.solver.get_reg(key)
            if prop_key == "clear":
                return lambda: obj.solver.regs.clear()
            return None
        return super().js_get_prop(obj, prop)


def solve_turnstile_dx(requirements_token: str, dx: str) -> str:
    return solve_turnstile_dx_with_session(requirements_token, dx, None)


def solve_turnstile_dx_with_session(requirements_token: str, dx: str, session: Session | None = None) -> str:
    return TurnstileSolver(requirements_token, dx, session=session).solve()


def solve_session_observer_with_proof(proof: str, collector_dx: str, session: Session | None = None) -> str:
    return SessionObserverSolver(proof, collector_dx, session=session).solve()


def object_keys(value: Any) -> list[Any]:
    if isinstance(value, dict):
        if "__storage_data__" in value and isinstance(value["__storage_data__"], dict):
            return list(keys_of_map(value["__storage_data__"]))
        if isinstance(value.get(ORDERED_KEYS_META), list):
            return [key for key in value[ORDERED_KEYS_META] if not is_internal_meta_key(key)]
        return [key for key in sorted(value.keys()) if not is_internal_meta_key(key)]
    if isinstance(value, list):
        return [float(i) for i in range(len(value))]
    return []


def keys_of_map(value: dict[str, Any]) -> list[str]:
    if isinstance(value.get(ORDERED_KEYS_META), list):
        return list(value[ORDERED_KEYS_META])
    return sorted(key for key in value.keys() if not is_internal_meta_key(key))


def with_ordered_keys(value: dict[str, Any] | None, keys: list[str]) -> dict[str, Any]:
    value = value or {}
    ordered: list[str] = []
    seen: set[str] = set()
    for key in keys:
        if is_internal_meta_key(key) or key in seen:
            continue
        ordered.append(key)
        seen.add(key)
    for key in sorted(value.keys()):
        if is_internal_meta_key(key) or key in seen:
            continue
        ordered.append(key)
        seen.add(key)
    value[ORDERED_KEYS_META] = ordered
    return value


def append_ordered_key(value: dict[str, Any], key: str) -> None:
    if is_internal_meta_key(key):
        return
    keys = value.get(ORDERED_KEYS_META)
    if not isinstance(keys, list):
        value[ORDERED_KEYS_META] = [key]
        return
    if key not in keys:
        keys.append(key)


def is_internal_meta_key(key: str) -> bool:
    return key in {ORDERED_KEYS_META, PROTOTYPE_META, "__storage_data__", "__storage_keys__"}


def js_json_stringify(value: Any) -> str:
    if value is None:
        return "null"
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, str):
        return json.dumps(value, ensure_ascii=False)
    if isinstance(value, int):
        return str(value)
    if isinstance(value, float):
        if math.isnan(value) or math.isinf(value):
            return "null"
        return json.dumps(value)
    if isinstance(value, list):
        return "[" + ",".join(js_json_stringify(item) for item in value) + "]"
    if isinstance(value, dict):
        parts = []
        for key in keys_of_map(value):
            if is_internal_meta_key(key):
                continue
            parts.append(json.dumps(key, ensure_ascii=False) + ":" + js_json_stringify(value.get(key)))
        return "{" + ",".join(parts) + "}"
    return json.dumps(value, ensure_ascii=False)


def string_slice_to_any(values: list[str]) -> list[Any]:
    return [item.strip() for item in values if item.strip()]


def json_string(value: Any) -> str:
    if value is None:
        return ""
    if isinstance(value, str):
        return value
    return str(value)


def json_float(value: Any) -> float:
    if isinstance(value, (int, float)):
        return float(value)
    if isinstance(value, str):
        try:
            return float(value.strip())
        except ValueError:
            return 0.0
    return 0.0


def reg_key(value: Any) -> str:
    if value is None:
        return "nil"
    if isinstance(value, str):
        return "s:" + value
    if isinstance(value, int):
        return f"n:{value}"
    if isinstance(value, float):
        return f"n:{int(value)}" if math.trunc(value) == value else "n:" + format(value, "g")
    return "x:" + str(value)


def to_int_index(value: Any) -> int:
    if isinstance(value, int):
        return value
    if isinstance(value, float) and math.trunc(value) == value:
        return int(value)
    if isinstance(value, str):
        try:
            return int(value.strip())
        except ValueError:
            return -1
    return -1


def copy_any_slice(value: list[Any]) -> list[Any]:
    return list(value or [])


def latin1_base64_decode(value: str) -> str:
    candidate = str(value or "").strip()
    if not candidate:
        return ""
    pad = "=" * ((4 - (len(candidate) % 4)) % 4)
    candidate = candidate + pad
    try:
        return base64.b64decode(candidate.encode("ascii")).decode("latin1")
    except Exception:
        return base64.urlsafe_b64decode(candidate.encode("ascii")).decode("latin1")


def latin1_base64_encode(value: str) -> str:
    return base64.b64encode(value.encode("latin1")).decode("ascii")


def xor_string(data: str, key: str) -> str:
    if not key:
        return data
    data_bytes = data.encode("latin1")
    key_bytes = key.encode("latin1")
    out = bytes(data_bytes[idx] ^ key_bytes[idx % len(key_bytes)] for idx in range(len(data_bytes)))
    return out.decode("latin1")
