var ALIYUN_ICON_CDN = function ALIYUN_ICON_CDN(type) {
  return "https://registry.npmmirror.com/@lobehub/icons-static-".concat(type, "/latest/files");
};
var UNPKG_ICON_CDN = function UNPKG_ICON_CDN(type) {
  return "https://unpkg.com/@lobehub/icons-static-".concat(type, "@latest");
};
export var getLobeIconCDN = function getLobeIconCDN(id, config) {
  var _ref = config || {},
    _ref$format = _ref.format,
    format = _ref$format === void 0 ? 'png' : _ref$format,
    _ref$isDarkMode = _ref.isDarkMode,
    isDarkMode = _ref$isDarkMode === void 0 ? false : _ref$isDarkMode,
    _ref$type = _ref.type,
    type = _ref$type === void 0 ? 'color' : _ref$type,
    _ref$cdn = _ref.cdn,
    cdn = _ref$cdn === void 0 ? 'aliyun' : _ref$cdn;
  var baseUrl = cdn === 'unpkg' ? UNPKG_ICON_CDN(format) : ALIYUN_ICON_CDN(format);
  var addon = type === 'mono' ? '' : "-".concat(type);
  switch (format) {
    case 'svg':
      {
        return "".concat(baseUrl, "/icons/").concat(id.toLowerCase() + addon, ".svg");
      }
    case 'webp':
      {
        return "".concat(baseUrl, "/").concat(isDarkMode ? 'dark' : 'light', "/").concat(id.toLowerCase() + addon, ".webp");
      }
    default:
      {
        return "".concat(baseUrl, "/").concat(isDarkMode ? 'dark' : 'light', "/").concat(id.toLowerCase() + addon, ".png");
      }
  }
};