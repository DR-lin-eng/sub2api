import { kebabCase } from 'es-toolkit';
import { useMemo } from 'react';
export var useFillId = function useFillId(namespace) {
  var id = "lobe-icons-".concat(kebabCase(namespace), "-fill");
  return useMemo(function () {
    return {
      fill: "url(#".concat(id, ")"),
      id: id
    };
  }, [namespace]);
};
export var useFillIds = function useFillIds(namespace, length) {
  return useMemo(function () {
    var ids = Array.from({
      length: length
    }, function (_, i) {
      var id = "lobe-icons-".concat(kebabCase(namespace), "-fill-").concat(i);
      return {
        fill: "url(#".concat(id, ")"),
        id: id
      };
    });
    return ids;
  }, [namespace, length]);
};