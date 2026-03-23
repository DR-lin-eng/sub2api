import { kebabCase } from 'es-toolkit';
export var customKebabCase = function customKebabCase(string) {
  var parts = string.match(/[A-Z]{2,}(?=[A-Z][a-z]+\d*|\b)|[A-Z]?[a-z]+\d*|[A-Z]|\d+/g);
  if (!parts) return string;
  return parts.map(function (part) {
    if (/\d/.test(part)) {
      return part.toLowerCase();
    }
    return kebabCase(part);
  }).join('-');
};