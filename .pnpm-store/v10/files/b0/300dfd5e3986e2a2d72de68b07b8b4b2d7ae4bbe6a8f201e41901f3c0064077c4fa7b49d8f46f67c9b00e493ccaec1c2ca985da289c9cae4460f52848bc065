"use strict";
'use client';

var _interopRequireWildcard = require("@babel/runtime/helpers/interopRequireWildcard").default;
Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.FieldLabel = void 0;
var React = _interopRequireWildcard(require("react"));
var _useIsoLayoutEffect = require("@base-ui/utils/useIsoLayoutEffect");
var _utils = require("../../floating-ui-react/utils");
var _FieldRootContext = require("../root/FieldRootContext");
var _LabelableContext = require("../../labelable-provider/LabelableContext");
var _constants = require("../utils/constants");
var _useBaseUiId = require("../../utils/useBaseUiId");
var _useRenderElement = require("../../utils/useRenderElement");
/**
 * An accessible label that is automatically associated with the field control.
 * Renders a `<label>` element.
 *
 * Documentation: [Base UI Field](https://base-ui.com/react/components/field)
 */
const FieldLabel = exports.FieldLabel = /*#__PURE__*/React.forwardRef(function FieldLabel(componentProps, forwardedRef) {
  const {
    render,
    className,
    id: idProp,
    ...elementProps
  } = componentProps;
  const fieldRootContext = (0, _FieldRootContext.useFieldRootContext)(false);
  const {
    controlId,
    setLabelId,
    labelId
  } = (0, _LabelableContext.useLabelableContext)();
  const id = (0, _useBaseUiId.useBaseUiId)(idProp);
  const labelRef = React.useRef(null);
  (0, _useIsoLayoutEffect.useIsoLayoutEffect)(() => {
    if (id) {
      setLabelId(id);
    }
    return () => {
      setLabelId(undefined);
    };
  }, [id, setLabelId]);
  const element = (0, _useRenderElement.useRenderElement)('label', componentProps, {
    ref: [forwardedRef, labelRef],
    state: fieldRootContext.state,
    props: [{
      id: labelId,
      htmlFor: controlId ?? undefined,
      onMouseDown(event) {
        const target = (0, _utils.getTarget)(event.nativeEvent);
        if (target?.closest('button,input,select,textarea')) {
          return;
        }

        // Prevent text selection when double clicking label.
        if (!event.defaultPrevented && event.detail > 1) {
          event.preventDefault();
        }
      }
    }, elementProps],
    stateAttributesMapping: _constants.fieldValidityMapping
  });
  return element;
});
if (process.env.NODE_ENV !== "production") FieldLabel.displayName = "FieldLabel";