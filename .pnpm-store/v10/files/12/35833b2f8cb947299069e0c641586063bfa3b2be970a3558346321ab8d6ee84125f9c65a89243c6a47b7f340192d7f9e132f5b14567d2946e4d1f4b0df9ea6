function _typeof(o) { "@babel/helpers - typeof"; return _typeof = "function" == typeof Symbol && "symbol" == typeof Symbol.iterator ? function (o) { return typeof o; } : function (o) { return o && "function" == typeof Symbol && o.constructor === Symbol && o !== Symbol.prototype ? "symbol" : typeof o; }, _typeof(o); }
var _excluded = ["size", "type"],
  _excluded2 = ["size", "type"],
  _excluded3 = ["size", "type"],
  _excluded4 = ["size", "type"],
  _excluded5 = ["size", "type"],
  _excluded6 = ["size", "type"],
  _excluded7 = ["size", "type"],
  _excluded8 = ["size"],
  _excluded9 = ["size"];
function ownKeys(e, r) { var t = Object.keys(e); if (Object.getOwnPropertySymbols) { var o = Object.getOwnPropertySymbols(e); r && (o = o.filter(function (r) { return Object.getOwnPropertyDescriptor(e, r).enumerable; })), t.push.apply(t, o); } return t; }
function _objectSpread(e) { for (var r = 1; r < arguments.length; r++) { var t = null != arguments[r] ? arguments[r] : {}; r % 2 ? ownKeys(Object(t), !0).forEach(function (r) { _defineProperty(e, r, t[r]); }) : Object.getOwnPropertyDescriptors ? Object.defineProperties(e, Object.getOwnPropertyDescriptors(t)) : ownKeys(Object(t)).forEach(function (r) { Object.defineProperty(e, r, Object.getOwnPropertyDescriptor(t, r)); }); } return e; }
function _defineProperty(obj, key, value) { key = _toPropertyKey(key); if (key in obj) { Object.defineProperty(obj, key, { value: value, enumerable: true, configurable: true, writable: true }); } else { obj[key] = value; } return obj; }
function _toPropertyKey(t) { var i = _toPrimitive(t, "string"); return "symbol" == _typeof(i) ? i : String(i); }
function _toPrimitive(t, r) { if ("object" != _typeof(t) || !t) return t; var e = t[Symbol.toPrimitive]; if (void 0 !== e) { var i = e.call(t, r || "default"); if ("object" != _typeof(i)) return i; throw new TypeError("@@toPrimitive must return a primitive value."); } return ("string" === r ? String : Number)(t); }
function _objectWithoutProperties(source, excluded) { if (source == null) return {}; var target = _objectWithoutPropertiesLoose(source, excluded); var key, i; if (Object.getOwnPropertySymbols) { var sourceSymbolKeys = Object.getOwnPropertySymbols(source); for (i = 0; i < sourceSymbolKeys.length; i++) { key = sourceSymbolKeys[i]; if (excluded.indexOf(key) >= 0) continue; if (!Object.prototype.propertyIsEnumerable.call(source, key)) continue; target[key] = source[key]; } } return target; }
function _objectWithoutPropertiesLoose(source, excluded) { if (source == null) return {}; var target = {}; var sourceKeys = Object.keys(source); var key, i; for (i = 0; i < sourceKeys.length; i++) { key = sourceKeys[i]; if (excluded.indexOf(key) >= 0) continue; target[key] = source[key]; } return target; }
import { memo } from 'react';
import Ai21 from "../Ai21";
import Ai302 from "../Ai302";
import Ai360 from "../Ai360";
import AiHubMix from "../AiHubMix";
import AiMass from "../AiMass";
import AkashChat from "../AkashChat";
import AlibabaCloud from "../AlibabaCloud";
import Anthropic from "../Anthropic";
import Aws from "../Aws";
import Azure from "../Azure";
import AzureAI from "../AzureAI";
import Baichuan from "../Baichuan";
import BaiduCloud from "../BaiduCloud";
import Bedrock from "../Bedrock";
import Bfl from "../Bfl";
import BurnCloud from "../BurnCloud";
import Cerebras from "../Cerebras";
import Claude from "../Claude";
import Cloudflare from "../Cloudflare";
import Cohere from "../Cohere";
import CometAPI from "../CometAPI";
import ComfyUI from "../ComfyUI";
import DeepSeek from "../DeepSeek";
import Doubao from "../Doubao";
import Fal from "../Fal";
import Fireworks from "../Fireworks";
import Gemini from "../Gemini";
import GiteeAI from "../GiteeAI";
import Github from "../Github";
import Google from "../Google";
import Groq from "../Groq";
import Higress from "../Higress";
import HuggingFace from "../HuggingFace";
import Hunyuan from "../Hunyuan";
import Infinigence from "../Infinigence";
import InternLM from "../InternLM";
import Jina from "../Jina";
import LmStudio from "../LmStudio";
import LobeHub from "../LobeHub";
import LongCat from "../LongCat";
import Minimax from "../Minimax";
import Mistral from "../Mistral";
import ModelScope from "../ModelScope";
import Moonshot from "../Moonshot";
import Nebius from "../Nebius";
import NewAPI from "../NewAPI";
import Novita from "../Novita";
import Nvidia from "../Nvidia";
import Ollama from "../Ollama";
import OpenAI from "../OpenAI";
import OpenRouter from "../OpenRouter";
import PPIO from "../PPIO";
import Perplexity from "../Perplexity";
import Player2 from "../Player2";
import Qiniu from "../Qiniu";
import Qwen from "../Qwen";
import Replicate from "../Replicate";
import SambaNova from "../SambaNova";
import Search1API from "../Search1API";
import SenseNova from "../SenseNova";
import SiliconCloud from "../SiliconCloud";
import SophNet from "../SophNet";
import Spark from "../Spark";
import Stepfun from "../Stepfun";
import Straico from "../Straico";
import TencentCloud from "../TencentCloud";
import Together from "../Together";
import Upstage from "../Upstage";
import V0 from "../V0";
import Vercel from "../Vercel";
import VertexAI from "../VertexAI";
import Vllm from "../Vllm";
import Volcengine from "../Volcengine";
import Wenxin from "../Wenxin";
import WorkersAI from "../WorkersAI";
import XAI from "../XAI";
import XiaomiMiMo from "../XiaomiMiMo";
import Xinference from "../Xinference";
import ZenMux from "../ZenMux";
import ZeroOne from "../ZeroOne";
import Zhipu from "../Zhipu";
import Combine from "./ProviderCombine/Combine";
import { ModelProvider } from "./providerEnum";
import { jsx as _jsx } from "react/jsx-runtime";
export var providerMappings = [{
  Icon: LobeHub,
  combineMultiple: 1.1,
  keywords: [ModelProvider.LobeHub]
}, {
  Icon: Zhipu,
  combineMultiple: 1.25,
  keywords: [ModelProvider.ZhiPu]
}, {
  Combine: /*#__PURE__*/memo(function (_ref) {
    var _ref$size = _ref.size,
      size = _ref$size === void 0 ? 24 : _ref$size,
      _ref$type = _ref.type,
      type = _ref$type === void 0 ? 'color' : _ref$type,
      props = _objectWithoutProperties(_ref, _excluded);
    return /*#__PURE__*/_jsx(Combine, _objectSpread({
      left: type === 'color' ? /*#__PURE__*/_jsx(Aws.Color, {
        size: size * 1.2
      }) : /*#__PURE__*/_jsx(Aws, {
        size: size * 1.2
      }),
      right: /*#__PURE__*/_jsx(Bedrock.Combine, {
        size: size,
        type: type
      }),
      size: size
    }, props));
  }),
  Icon: Bedrock,
  combineMultiple: 1.1,
  keywords: [ModelProvider.Bedrock]
}, {
  Icon: DeepSeek,
  combineMultiple: 1.16,
  keywords: [ModelProvider.DeepSeek]
}, {
  Combine: /*#__PURE__*/memo(function (_ref2) {
    var _ref2$size = _ref2.size,
      size = _ref2$size === void 0 ? 24 : _ref2$size,
      _ref2$type = _ref2.type,
      type = _ref2$type === void 0 ? 'color' : _ref2$type,
      props = _objectWithoutProperties(_ref2, _excluded2);
    return /*#__PURE__*/_jsx(Combine, _objectSpread({
      left: type === 'color' ? /*#__PURE__*/_jsx(Google.BrandColor, {
        size: size * 0.95
      }) : /*#__PURE__*/_jsx(Google.Brand, {
        size: size * 0.95
      }),
      right: /*#__PURE__*/_jsx(Gemini.Combine, {
        size: size,
        type: type
      }),
      size: size
    }, props));
  }),
  Icon: Google,
  combineMultiple: 0.92,
  keywords: [ModelProvider.Google]
}, {
  Combine: /*#__PURE__*/memo(function (_ref3) {
    var _ref3$size = _ref3.size,
      size = _ref3$size === void 0 ? 24 : _ref3$size,
      _ref3$type = _ref3.type,
      type = _ref3$type === void 0 ? 'color' : _ref3$type,
      props = _objectWithoutProperties(_ref3, _excluded3);
    return /*#__PURE__*/_jsx(Combine, _objectSpread({
      left: /*#__PURE__*/_jsx(Azure.Combine, {
        size: size * 0.92,
        type: type
      }),
      right: /*#__PURE__*/_jsx(OpenAI.Combine, {
        size: size
      }),
      size: size
    }, props));
  }),
  Icon: Azure,
  combineMultiple: 0.9,
  keywords: [ModelProvider.Azure]
}, {
  Icon: Moonshot,
  combineMultiple: 0.9,
  keywords: [ModelProvider.Moonshot]
}, {
  Icon: Novita,
  keywords: [ModelProvider.Novita]
}, {
  Icon: OpenAI,
  keywords: [ModelProvider.OpenAI]
}, {
  Icon: Ollama,
  combineMultiple: 1.16,
  keywords: [ModelProvider.Ollama]
}, {
  Icon: Perplexity,
  keywords: [ModelProvider.Perplexity]
}, {
  Icon: Minimax,
  combineMultiple: 1.3,
  keywords: [ModelProvider.Minimax]
}, {
  Icon: Mistral,
  keywords: [ModelProvider.Mistral]
}, {
  Combine: /*#__PURE__*/memo(function (_ref4) {
    var _ref4$size = _ref4.size,
      size = _ref4$size === void 0 ? 24 : _ref4$size,
      _ref4$type = _ref4.type,
      type = _ref4$type === void 0 ? 'color' : _ref4$type,
      props = _objectWithoutProperties(_ref4, _excluded4);
    return /*#__PURE__*/_jsx(Combine, _objectSpread({
      left: /*#__PURE__*/_jsx(Anthropic.Text, {
        size: size * 0.75
      }),
      right: /*#__PURE__*/_jsx(Claude.Combine, {
        size: size,
        type: type
      }),
      size: size
    }, props));
  }),
  Icon: Anthropic,
  combineMultiple: 0.83,
  keywords: [ModelProvider.Anthropic]
}, {
  Icon: Groq,
  keywords: [ModelProvider.Groq]
}, {
  Icon: OpenRouter,
  combineMultiple: 0.8,
  keywords: [ModelProvider.OpenRouter]
}, {
  Icon: ZeroOne,
  combineMultiple: 1,
  keywords: [ModelProvider.ZeroOne]
}, {
  Icon: Together,
  keywords: [ModelProvider.TogetherAI]
}, {
  Icon: Qiniu,
  combineMultiple: 1.1,
  keywords: [ModelProvider.Qiniu]
}, {
  Combine: /*#__PURE__*/memo(function (_ref5) {
    var _ref5$size = _ref5.size,
      size = _ref5$size === void 0 ? 24 : _ref5$size,
      _ref5$type = _ref5.type,
      type = _ref5$type === void 0 ? 'color' : _ref5$type,
      props = _objectWithoutProperties(_ref5, _excluded5);
    return /*#__PURE__*/_jsx(Combine, _objectSpread({
      left: /*#__PURE__*/_jsx(AlibabaCloud.Combine, {
        size: size,
        type: type
      }),
      right: /*#__PURE__*/_jsx(Qwen.Combine, {
        size: size * 0.9,
        type: type
      }),
      size: size
    }, props));
  }),
  Icon: AlibabaCloud,
  combineMultiple: 1.1,
  keywords: [ModelProvider.Qwen]
}, {
  Icon: Stepfun,
  combineMultiple: 0.83,
  keywords: [ModelProvider.Stepfun]
}, {
  Icon: Spark,
  combineMultiple: 0.92,
  keywords: [ModelProvider.Spark]
}, {
  Icon: Fireworks,
  combineMultiple: 1.14,
  keywords: [ModelProvider.FireworksAI]
}, {
  Icon: Baichuan,
  combineMultiple: 0.83,
  keywords: [ModelProvider.Baichuan]
}, {
  Icon: BurnCloud,
  combineMultiple: 1.2,
  keywords: [ModelProvider.BurnCloud]
}, {
  Icon: AiMass,
  combineMultiple: 1.16,
  keywords: [ModelProvider.Taichu]
}, {
  Icon: Ai360,
  combineMultiple: 0.83,
  keywords: [ModelProvider.Ai360]
}, {
  Icon: SiliconCloud,
  combineMultiple: 1,
  keywords: [ModelProvider.SiliconCloud]
}, {
  Icon: Upstage,
  combineMultiple: 0.9,
  keywords: [ModelProvider.Upstage]
}, {
  Icon: Ai21,
  combineMultiple: 0.9,
  keywords: [ModelProvider.Ai21]
}, {
  Icon: Player2,
  combineMultiple: 0.9,
  keywords: [ModelProvider.Player2]
}, {
  Icon: Github,
  combineMultiple: 0.95,
  keywords: [ModelProvider.Github]
}, {
  Icon: Doubao,
  keywords: [ModelProvider.Doubao]
}, {
  Icon: Hunyuan,
  keywords: [ModelProvider.Hunyuan]
}, {
  Icon: Nvidia,
  keywords: [ModelProvider.Nvidia]
}, {
  Icon: TencentCloud,
  keywords: [ModelProvider.TencentCloud]
}, {
  Combine: /*#__PURE__*/memo(function (_ref6) {
    var _ref6$size = _ref6.size,
      size = _ref6$size === void 0 ? 24 : _ref6$size,
      _ref6$type = _ref6.type,
      type = _ref6$type === void 0 ? 'color' : _ref6$type,
      props = _objectWithoutProperties(_ref6, _excluded6);
    return /*#__PURE__*/_jsx(Combine, _objectSpread({
      left: /*#__PURE__*/_jsx(BaiduCloud.Combine, {
        size: size * 0.9,
        type: type
      }),
      right: /*#__PURE__*/_jsx(Wenxin.Combine, _objectSpread({
        extra: '千帆',
        size: size,
        type: type
      }, props)),
      size: size
    }, props));
  }),
  Icon: Wenxin,
  keywords: [ModelProvider.Wenxin]
}, {
  Icon: SenseNova,
  combineMultiple: 0.95,
  keywords: [ModelProvider.SenseNova]
}, {
  Icon: HuggingFace,
  combineMultiple: 1.16,
  keywords: [ModelProvider.HuggingFace]
}, {
  Icon: LmStudio,
  keywords: [ModelProvider.LmStudio]
}, {
  Icon: XAI,
  combineMultiple: 0.85,
  keywords: [ModelProvider.XAI]
}, {
  Combine: /*#__PURE__*/memo(function (_ref7) {
    var _ref7$size = _ref7.size,
      size = _ref7$size === void 0 ? 24 : _ref7$size,
      _ref7$type = _ref7.type,
      type = _ref7$type === void 0 ? 'color' : _ref7$type,
      props = _objectWithoutProperties(_ref7, _excluded7);
    return /*#__PURE__*/_jsx(Combine, _objectSpread({
      left: /*#__PURE__*/_jsx(Cloudflare.Combine, {
        size: size * 1.1,
        type: type
      }),
      right: /*#__PURE__*/_jsx(WorkersAI.Combine, {
        size: size * 0.9,
        type: type
      }),
      size: size
    }, props));
  }),
  Icon: Cloudflare,
  combineMultiple: 1.1,
  keywords: [ModelProvider.Cloudflare]
}, {
  Icon: InternLM,
  combineMultiple: 0.95,
  keywords: [ModelProvider.InternLM]
}, {
  Icon: Higress,
  keywords: [ModelProvider.Higress]
}, {
  Icon: Vllm,
  combineMultiple: 0.85,
  keywords: [ModelProvider.VLLM]
}, {
  Icon: GiteeAI,
  combineMultiple: 0.95,
  keywords: [ModelProvider.GiteeAI]
}, {
  Icon: ModelScope,
  combineMultiple: 1.2,
  keywords: [ModelProvider.ModelScope]
}, {
  Icon: VertexAI,
  keywords: [ModelProvider.VertexAI]
}, {
  Icon: PPIO,
  combineMultiple: 0.85,
  keywords: [ModelProvider.PPIO]
}, {
  Icon: Jina,
  keywords: [ModelProvider.Jina]
}, {
  Icon: AzureAI,
  keywords: [ModelProvider.AzureAI]
}, {
  Icon: Volcengine,
  keywords: [ModelProvider.Volcengine]
}, {
  Icon: SambaNova,
  combineMultiple: 0.8,
  keywords: [ModelProvider.SambaNova]
}, {
  Icon: Cohere,
  keywords: [ModelProvider.Cohere]
}, {
  Icon: ComfyUI,
  keywords: [ModelProvider.ComfyUI]
}, {
  Icon: Search1API,
  combineMultiple: 0.9,
  keywords: [ModelProvider.Search1API]
}, {
  Icon: Infinigence,
  combineMultiple: 0.8,
  keywords: [ModelProvider.InfiniAI]
}, {
  Icon: Xinference,
  combineMultiple: 0.85,
  keywords: [ModelProvider.Xinference]
}, {
  Icon: Fal,
  combineMultiple: 0.8,
  keywords: [ModelProvider.Fal]
}, {
  Icon: Ai302,
  combineMultiple: 0.9,
  keywords: [ModelProvider.Ai302]
}, {
  Icon: AiHubMix,
  combineMultiple: 0.9,
  keywords: [ModelProvider.AiHubMix]
}, {
  Icon: CometAPI,
  keywords: [ModelProvider.CometAPI]
}, {
  Combine: /*#__PURE__*/memo(function (_ref8) {
    var _ref8$size = _ref8.size,
      size = _ref8$size === void 0 ? 24 : _ref8$size,
      props = _objectWithoutProperties(_ref8, _excluded8);
    return /*#__PURE__*/_jsx(Combine, _objectSpread({
      left: /*#__PURE__*/_jsx(Vercel.Combine, {
        size: size * 0.85
      }),
      right: /*#__PURE__*/_jsx(V0, {
        size: size * 1.1
      }),
      size: size
    }, props));
  }),
  Icon: Vercel,
  keywords: [ModelProvider.V0]
}, {
  Icon: Vercel,
  combineMultiple: 0.85,
  keywords: [ModelProvider.Vercel]
}, {
  Icon: Bfl,
  keywords: [ModelProvider.Bfl]
}, {
  Icon: Replicate,
  combineMultiple: 0.9,
  keywords: [ModelProvider.Replicate]
}, {
  Icon: Nebius,
  combineMultiple: 0.75,
  keywords: [ModelProvider.Nebius]
}, {
  Icon: NewAPI,
  combineMultiple: 0.85,
  keywords: [ModelProvider.NewAPI]
}, {
  Icon: AkashChat,
  combineMultiple: 0.8,
  keywords: [ModelProvider.AkashChat]
}, {
  Icon: SophNet,
  combineMultiple: 0.85,
  keywords: [ModelProvider.SophNet]
}, {
  Combine: /*#__PURE__*/memo(function (_ref9) {
    var _ref9$size = _ref9.size,
      size = _ref9$size === void 0 ? 24 : _ref9$size,
      props = _objectWithoutProperties(_ref9, _excluded9);
    return /*#__PURE__*/_jsx(Ollama.Combine, _objectSpread({
      extra: 'Cloud',
      extraStyle: {
        fontSize: size * 0.78,
        fontWeight: 500,
        marginLeft: size * 0.2
      },
      size: size * 1.16
    }, props));
  }),
  Icon: Ollama,
  keywords: [ModelProvider.OllamaCloud]
}, {
  Icon: LongCat,
  combineMultiple: 1,
  keywords: [ModelProvider.LongCat]
}, {
  Icon: Cerebras,
  combineMultiple: 1,
  keywords: [ModelProvider.Cerebras]
}, {
  Icon: Straico,
  combineMultiple: 0.85,
  keywords: [ModelProvider.Straico]
}, {
  Icon: ZenMux,
  combineMultiple: 1,
  keywords: [ModelProvider.ZenMux],
  props: {
    inverse: true
  }
}, {
  Icon: XiaomiMiMo,
  combineMultiple: 0.7,
  keywords: [ModelProvider.XiaomiMiMo]
}];