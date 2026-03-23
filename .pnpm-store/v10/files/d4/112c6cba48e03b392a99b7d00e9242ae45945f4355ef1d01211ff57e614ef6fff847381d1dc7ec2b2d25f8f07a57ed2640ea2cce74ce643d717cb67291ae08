"use strict";
'use client';

var _interopRequireWildcard = require("@babel/runtime/helpers/interopRequireWildcard").default;
Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.SelectTrigger = void 0;
var React = _interopRequireWildcard(require("react"));
var _owner = require("@base-ui/utils/owner");
var _useTimeout = require("@base-ui/utils/useTimeout");
var _useStableCallback = require("@base-ui/utils/useStableCallback");
var _useMergedRefs = require("@base-ui/utils/useMergedRefs");
var _useValueAsRef = require("@base-ui/utils/useValueAsRef");
var _store = require("@base-ui/utils/store");
var _SelectRootContext = require("../root/SelectRootContext");
var _FieldRootContext = require("../../field/root/FieldRootContext");
var _LabelableContext = require("../../labelable-provider/LabelableContext");
var _popupStateMapping = require("../../utils/popupStateMapping");
var _constants = require("../../field/utils/constants");
var _useRenderElement = require("../../utils/useRenderElement");
var _store2 = require("../store");
var _getPseudoElementBounds = require("../../utils/getPseudoElementBounds");
var _utils = require("../../floating-ui-react/utils");
var _mergeProps = require("../../merge-props");
var _useButton = require("../../use-button");
var _createBaseUIEventDetails = require("../../utils/createBaseUIEventDetails");
var _reasons = require("../../utils/reasons");
const BOUNDARY_OFFSET = 2;
const stateAttributesMapping = {
  ..._popupStateMapping.pressableTriggerOpenStateMapping,
  ..._constants.fieldValidityMapping,
  value: () => null
};

/**
 * A button that opens the select popup.
 * Renders a `<button>` element.
 *
 * Documentation: [Base UI Select](https://base-ui.com/react/components/select)
 */
const SelectTrigger = exports.SelectTrigger = /*#__PURE__*/React.forwardRef(function SelectTrigger(componentProps, forwardedRef) {
  const {
    render,
    className,
    disabled: disabledProp = false,
    nativeButton = true,
    ...elementProps
  } = componentProps;
  const {
    setTouched,
    setFocused,
    validationMode,
    state: fieldState,
    disabled: fieldDisabled
  } = (0, _FieldRootContext.useFieldRootContext)();
  const {
    labelId
  } = (0, _LabelableContext.useLabelableContext)();
  const {
    store,
    setOpen,
    selectionRef,
    validation,
    readOnly,
    alignItemWithTriggerActiveRef,
    disabled: selectDisabled,
    keyboardActiveRef
  } = (0, _SelectRootContext.useSelectRootContext)();
  const disabled = fieldDisabled || selectDisabled || disabledProp;
  const open = (0, _store.useStore)(store, _store2.selectors.open);
  const value = (0, _store.useStore)(store, _store2.selectors.value);
  const triggerProps = (0, _store.useStore)(store, _store2.selectors.triggerProps);
  const positionerElement = (0, _store.useStore)(store, _store2.selectors.positionerElement);
  const listElement = (0, _store.useStore)(store, _store2.selectors.listElement);
  const serializedValue = (0, _store.useStore)(store, _store2.selectors.serializedValue);
  const positionerRef = (0, _useValueAsRef.useValueAsRef)(positionerElement);
  const triggerRef = React.useRef(null);
  const timeoutFocus = (0, _useTimeout.useTimeout)();
  const timeoutMouseDown = (0, _useTimeout.useTimeout)();
  const {
    getButtonProps,
    buttonRef
  } = (0, _useButton.useButton)({
    disabled,
    native: nativeButton
  });
  const setTriggerElement = (0, _useStableCallback.useStableCallback)(element => {
    store.set('triggerElement', element);
  });
  const mergedRef = (0, _useMergedRefs.useMergedRefs)(forwardedRef, triggerRef, buttonRef, setTriggerElement);
  const timeout1 = (0, _useTimeout.useTimeout)();
  const timeout2 = (0, _useTimeout.useTimeout)();
  React.useEffect(() => {
    if (open) {
      // mousedown -> move to unselected item -> mouseup should not select within 200ms.
      timeout2.start(200, () => {
        selectionRef.current.allowUnselectedMouseUp = true;

        // mousedown -> mouseup on selected item should not select within 400ms.
        timeout1.start(200, () => {
          selectionRef.current.allowSelectedMouseUp = true;
        });
      });
      return () => {
        timeout1.clear();
        timeout2.clear();
      };
    }
    selectionRef.current = {
      allowSelectedMouseUp: false,
      allowUnselectedMouseUp: false
    };
    timeoutMouseDown.clear();
    return undefined;
  }, [open, selectionRef, timeoutMouseDown, timeout1, timeout2]);
  const ariaControlsId = React.useMemo(() => {
    return listElement?.id ?? (0, _utils.getFloatingFocusElement)(positionerElement)?.id;
  }, [listElement, positionerElement]);
  const props = (0, _mergeProps.mergeProps)(triggerProps, {
    role: 'combobox',
    'aria-expanded': open ? 'true' : 'false',
    'aria-haspopup': 'listbox',
    'aria-controls': open ? ariaControlsId : undefined,
    'aria-labelledby': labelId,
    'aria-readonly': readOnly || undefined,
    tabIndex: disabled ? -1 : 0,
    ref: mergedRef,
    onFocus(event) {
      setFocused(true);
      // The popup element shouldn't obscure the focused trigger.
      if (open && alignItemWithTriggerActiveRef.current) {
        setOpen(false, (0, _createBaseUIEventDetails.createChangeEventDetails)(_reasons.REASONS.focusOut, event.nativeEvent));
      }

      // Saves a re-render on initial click: `forceMount === true` mounts
      // the items before `open === true`. We could sync those cycles better
      // without a timeout, but this is enough for now.
      //
      // XXX: might be causing `act()` warnings.
      timeoutFocus.start(0, () => {
        store.set('forceMount', true);
      });
    },
    onBlur() {
      setTouched(true);
      setFocused(false);
      if (validationMode === 'onBlur') {
        validation.commit(value);
      }
    },
    onPointerMove() {
      keyboardActiveRef.current = false;
    },
    onKeyDown() {
      keyboardActiveRef.current = true;
    },
    onMouseDown(event) {
      if (open) {
        return;
      }
      const doc = (0, _owner.ownerDocument)(event.currentTarget);
      function handleMouseUp(mouseEvent) {
        if (!triggerRef.current) {
          return;
        }
        const mouseUpTarget = mouseEvent.target;

        // Early return if clicked on trigger element or its children
        if ((0, _utils.contains)(triggerRef.current, mouseUpTarget) || (0, _utils.contains)(positionerRef.current, mouseUpTarget) || mouseUpTarget === triggerRef.current) {
          return;
        }
        const bounds = (0, _getPseudoElementBounds.getPseudoElementBounds)(triggerRef.current);
        if (mouseEvent.clientX >= bounds.left - BOUNDARY_OFFSET && mouseEvent.clientX <= bounds.right + BOUNDARY_OFFSET && mouseEvent.clientY >= bounds.top - BOUNDARY_OFFSET && mouseEvent.clientY <= bounds.bottom + BOUNDARY_OFFSET) {
          return;
        }
        setOpen(false, (0, _createBaseUIEventDetails.createChangeEventDetails)(_reasons.REASONS.cancelOpen, mouseEvent));
      }

      // Firefox can fire this upon mousedown
      timeoutMouseDown.start(0, () => {
        doc.addEventListener('mouseup', handleMouseUp, {
          once: true
        });
      });
    }
  }, validation.getValidationProps, elementProps, getButtonProps);

  // ensure nested useButton does not overwrite the combobox role:
  // <Toolbar.Button render={<Select.Trigger />} />
  props.role = 'combobox';
  const state = React.useMemo(() => ({
    ...fieldState,
    open,
    disabled,
    value,
    readOnly,
    placeholder: !serializedValue
  }), [fieldState, open, disabled, value, readOnly, serializedValue]);
  return (0, _useRenderElement.useRenderElement)('button', componentProps, {
    ref: [forwardedRef, triggerRef],
    state,
    stateAttributesMapping,
    props
  });
});
if (process.env.NODE_ENV !== "production") SelectTrigger.displayName = "SelectTrigger";