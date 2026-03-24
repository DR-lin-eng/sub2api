"use strict";
'use client';

var _interopRequireWildcard = require("@babel/runtime/helpers/interopRequireWildcard").default;
Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.ComboboxTrigger = void 0;
var React = _interopRequireWildcard(require("react"));
var _store = require("@base-ui/utils/store");
var _useStableCallback = require("@base-ui/utils/useStableCallback");
var _useTimeout = require("@base-ui/utils/useTimeout");
var _owner = require("@base-ui/utils/owner");
var _useRenderElement = require("../../utils/useRenderElement");
var _useButton = require("../../use-button");
var _ComboboxRootContext = require("../root/ComboboxRootContext");
var _store2 = require("../store");
var _FieldRootContext = require("../../field/root/FieldRootContext");
var _LabelableContext = require("../../labelable-provider/LabelableContext");
var _popupStateMapping = require("../../utils/popupStateMapping");
var _utils = require("../../floating-ui-react/utils");
var _getPseudoElementBounds = require("../../utils/getPseudoElementBounds");
var _createBaseUIEventDetails = require("../../utils/createBaseUIEventDetails");
var _reasons = require("../../utils/reasons");
var _constants = require("../../field/utils/constants");
var _floatingUiReact = require("../../floating-ui-react");
const BOUNDARY_OFFSET = 2;
const stateAttributesMapping = {
  ..._popupStateMapping.pressableTriggerOpenStateMapping,
  ..._constants.fieldValidityMapping
};

/**
 * A button that opens the popup.
 * Renders a `<button>` element.
 */
const ComboboxTrigger = exports.ComboboxTrigger = /*#__PURE__*/React.forwardRef(function ComboboxTrigger(componentProps, forwardedRef) {
  const {
    render,
    className,
    nativeButton = true,
    disabled: disabledProp = false,
    ...elementProps
  } = componentProps;
  const {
    state: fieldState,
    disabled: fieldDisabled,
    setTouched,
    setFocused,
    validationMode,
    validation
  } = (0, _FieldRootContext.useFieldRootContext)();
  const {
    labelId
  } = (0, _LabelableContext.useLabelableContext)();
  const store = (0, _ComboboxRootContext.useComboboxRootContext)();
  const selectionMode = (0, _store.useStore)(store, _store2.selectors.selectionMode);
  const comboboxDisabled = (0, _store.useStore)(store, _store2.selectors.disabled);
  const readOnly = (0, _store.useStore)(store, _store2.selectors.readOnly);
  const listElement = (0, _store.useStore)(store, _store2.selectors.listElement);
  const triggerProps = (0, _store.useStore)(store, _store2.selectors.triggerProps);
  const triggerElement = (0, _store.useStore)(store, _store2.selectors.triggerElement);
  const inputInsidePopup = (0, _store.useStore)(store, _store2.selectors.inputInsidePopup);
  const open = (0, _store.useStore)(store, _store2.selectors.open);
  const selectedValue = (0, _store.useStore)(store, _store2.selectors.selectedValue);
  const activeIndex = (0, _store.useStore)(store, _store2.selectors.activeIndex);
  const selectedIndex = (0, _store.useStore)(store, _store2.selectors.selectedIndex);
  const floatingRootContext = (0, _ComboboxRootContext.useComboboxFloatingContext)();
  const inputValue = (0, _ComboboxRootContext.useComboboxInputValueContext)();
  const focusTimeout = (0, _useTimeout.useTimeout)();
  const disabled = fieldDisabled || comboboxDisabled || disabledProp;
  const currentPointerTypeRef = React.useRef('');
  function trackPointerType(event) {
    currentPointerTypeRef.current = event.pointerType;
  }
  const domReference = floatingRootContext.select('domReferenceElement');

  // Update the floating root context to use the trigger element when it differs from the current reference.
  // This ensures useClick and useTypeahead attach handlers to the correct element.
  React.useEffect(() => {
    if (!inputInsidePopup) {
      return;
    }
    if (triggerElement && triggerElement !== domReference) {
      floatingRootContext.set('domReferenceElement', triggerElement);
    }
  }, [triggerElement, domReference, floatingRootContext, inputInsidePopup]);
  const {
    reference: triggerTypeaheadProps
  } = (0, _floatingUiReact.useTypeahead)(floatingRootContext, {
    enabled: !open && !readOnly && !comboboxDisabled && selectionMode === 'single',
    listRef: store.state.labelsRef,
    activeIndex,
    selectedIndex,
    onMatch(index) {
      const nextSelectedValue = store.state.valuesRef.current[index];
      if (nextSelectedValue !== undefined) {
        store.state.setSelectedValue(nextSelectedValue, (0, _createBaseUIEventDetails.createChangeEventDetails)('none'));
      }
    }
  });
  const {
    reference: triggerClickProps
  } = (0, _floatingUiReact.useClick)(floatingRootContext, {
    enabled: !readOnly && !comboboxDisabled,
    event: 'mousedown'
  });
  const {
    buttonRef,
    getButtonProps
  } = (0, _useButton.useButton)({
    native: nativeButton,
    disabled
  });
  const state = React.useMemo(() => ({
    ...fieldState,
    open,
    disabled
  }), [fieldState, open, disabled]);
  const setTriggerElement = (0, _useStableCallback.useStableCallback)(element => {
    store.set('triggerElement', element);
  });
  const element = (0, _useRenderElement.useRenderElement)('button', componentProps, {
    ref: [forwardedRef, buttonRef, setTriggerElement],
    state,
    props: [triggerProps, triggerClickProps, triggerTypeaheadProps, {
      tabIndex: inputInsidePopup ? 0 : -1,
      disabled,
      role: inputInsidePopup ? 'combobox' : undefined,
      'aria-expanded': open ? 'true' : 'false',
      'aria-haspopup': inputInsidePopup ? 'dialog' : 'listbox',
      'aria-controls': open ? listElement?.id : undefined,
      'aria-readonly': readOnly || undefined,
      'aria-labelledby': labelId,
      onPointerDown: trackPointerType,
      onPointerEnter: trackPointerType,
      onFocus() {
        setFocused(true);
        if (disabled || readOnly) {
          return;
        }
        focusTimeout.start(0, store.state.forceMount);
      },
      onBlur() {
        setTouched(true);
        setFocused(false);
        if (validationMode === 'onBlur') {
          const valueToValidate = selectionMode === 'none' ? inputValue : selectedValue;
          validation.commit(valueToValidate);
        }
      },
      onMouseDown(event) {
        if (disabled || readOnly) {
          return;
        }
        if (!inputInsidePopup) {
          floatingRootContext.set('domReferenceElement', event.currentTarget);
        }

        // Ensure items are registered for initial selection highlight.
        store.state.forceMount();
        if (currentPointerTypeRef.current !== 'touch') {
          store.state.inputRef.current?.focus();
          if (!inputInsidePopup) {
            event.preventDefault();
          }
        }
        if (open) {
          return;
        }
        const doc = (0, _owner.ownerDocument)(event.currentTarget);
        function handleMouseUp(mouseEvent) {
          if (!triggerElement) {
            return;
          }
          const mouseUpTarget = (0, _utils.getTarget)(mouseEvent);
          const positioner = store.state.positionerElement;
          const list = store.state.listElement;
          if ((0, _utils.contains)(triggerElement, mouseUpTarget) || (0, _utils.contains)(positioner, mouseUpTarget) || (0, _utils.contains)(list, mouseUpTarget) || mouseUpTarget === triggerElement) {
            return;
          }
          const bounds = (0, _getPseudoElementBounds.getPseudoElementBounds)(triggerElement);
          const withinHorizontal = mouseEvent.clientX >= bounds.left - BOUNDARY_OFFSET && mouseEvent.clientX <= bounds.right + BOUNDARY_OFFSET;
          const withinVertical = mouseEvent.clientY >= bounds.top - BOUNDARY_OFFSET && mouseEvent.clientY <= bounds.bottom + BOUNDARY_OFFSET;
          if (withinHorizontal && withinVertical) {
            return;
          }
          store.state.setOpen(false, (0, _createBaseUIEventDetails.createChangeEventDetails)('cancel-open', mouseEvent));
        }
        if (inputInsidePopup) {
          doc.addEventListener('mouseup', handleMouseUp, {
            once: true
          });
        }
      },
      onKeyDown(event) {
        if (disabled || readOnly) {
          return;
        }
        if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
          (0, _utils.stopEvent)(event);
          store.state.setOpen(true, (0, _createBaseUIEventDetails.createChangeEventDetails)(_reasons.REASONS.listNavigation, event.nativeEvent));
          store.state.inputRef.current?.focus();
        }
      }
    }, validation ? validation.getValidationProps(elementProps) : elementProps, getButtonProps],
    stateAttributesMapping
  });
  return element;
});
if (process.env.NODE_ENV !== "production") ComboboxTrigger.displayName = "ComboboxTrigger";