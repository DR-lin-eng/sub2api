"use strict";

var _interopRequireWildcard = require("@babel/runtime/helpers/interopRequireWildcard").default;
Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.usePopupAutoResize = usePopupAutoResize;
var React = _interopRequireWildcard(require("react"));
var _useAnimationFrame = require("@base-ui/utils/useAnimationFrame");
var _useIsoLayoutEffect = require("@base-ui/utils/useIsoLayoutEffect");
var _useStableCallback = require("@base-ui/utils/useStableCallback");
var _useAnimationsFinished = require("./useAnimationsFinished");
var _getCssDimensions = require("./getCssDimensions");
var _constants = require("./constants");
const supportsResizeObserver = typeof ResizeObserver !== 'undefined';
const DEFAULT_ENABLED = () => true;

/**
 * Allows the element to automatically resize based on its content while supporting animations.
 */
function usePopupAutoResize(parameters) {
  const {
    popupElement,
    positionerElement,
    content,
    mounted,
    enabled = DEFAULT_ENABLED,
    onMeasureLayout: onMeasureLayoutParam,
    onMeasureLayoutComplete: onMeasureLayoutCompleteParam,
    side,
    direction
  } = parameters;
  const isInitialRender = React.useRef(true);
  const runOnceAnimationsFinish = (0, _useAnimationsFinished.useAnimationsFinished)(popupElement, true, false);
  const animationFrame = (0, _useAnimationFrame.useAnimationFrame)();
  const previousDimensionsRef = React.useRef(null);
  const onMeasureLayout = (0, _useStableCallback.useStableCallback)(onMeasureLayoutParam);
  const onMeasureLayoutComplete = (0, _useStableCallback.useStableCallback)(onMeasureLayoutCompleteParam);
  const anchoringStyles = React.useMemo(() => {
    // Ensure popup size transitions correctly when anchored to `bottom` (side=top) or `right` (side=left).
    let isOriginSide = side === 'top';
    let isPhysicalLeft = side === 'left';
    if (direction === 'rtl') {
      isOriginSide = isOriginSide || side === 'inline-end';
      isPhysicalLeft = isPhysicalLeft || side === 'inline-end';
    } else {
      isOriginSide = isOriginSide || side === 'inline-start';
      isPhysicalLeft = isPhysicalLeft || side === 'inline-start';
    }
    return isOriginSide ? {
      position: 'absolute',
      [side === 'top' ? 'bottom' : 'top']: '0',
      [isPhysicalLeft ? 'right' : 'left']: '0'
    } : _constants.EMPTY_OBJECT;
  }, [side, direction]);
  (0, _useIsoLayoutEffect.useIsoLayoutEffect)(() => {
    // Reset the state when the popup is closed.
    if (!mounted || !enabled() || !supportsResizeObserver) {
      isInitialRender.current = true;
      previousDimensionsRef.current = null;
      return undefined;
    }
    if (!popupElement || !positionerElement) {
      return undefined;
    }
    Object.entries(anchoringStyles).forEach(([key, value]) => {
      popupElement.style.setProperty(key, value);
    });
    const observer = new ResizeObserver(entries => {
      const entry = entries[0];
      if (entry) {
        if (previousDimensionsRef.current === null) {
          previousDimensionsRef.current = {
            width: Math.ceil(entry.borderBoxSize[0].inlineSize),
            height: Math.ceil(entry.borderBoxSize[0].blockSize)
          };
        } else {
          previousDimensionsRef.current.width = Math.ceil(entry.borderBoxSize[0].inlineSize);
          previousDimensionsRef.current.height = Math.ceil(entry.borderBoxSize[0].blockSize);
        }
      }
    });
    observer.observe(popupElement);

    // Measure the rendered size to enable transitions:
    popupElement.style.setProperty('--popup-width', 'auto');
    popupElement.style.setProperty('--popup-height', 'auto');
    const restorePopupPosition = overrideElementStyle(popupElement, 'position', 'static');
    const restorePopupTransform = overrideElementStyle(popupElement, 'transform', 'none');
    const restorePopupScale = overrideElementStyle(popupElement, 'scale', '1');
    const restoreAvailableWidth = overrideElementStyle(positionerElement, '--available-width', 'max-content');
    const restoreAvailableHeight = overrideElementStyle(positionerElement, '--available-height', 'max-content');
    onMeasureLayout?.();

    // Initial render (for each time the popup opens).
    if (isInitialRender.current || previousDimensionsRef.current === null) {
      positionerElement.style.setProperty('--positioner-width', 'max-content');
      positionerElement.style.setProperty('--positioner-height', 'max-content');
      const dimensions = (0, _getCssDimensions.getCssDimensions)(popupElement);
      positionerElement.style.setProperty('--positioner-width', `${dimensions.width}px`);
      positionerElement.style.setProperty('--positioner-height', `${dimensions.height}px`);
      restorePopupPosition();
      restorePopupTransform();
      restorePopupScale();
      restoreAvailableWidth();
      restoreAvailableHeight();
      onMeasureLayoutComplete?.(null, dimensions);
      isInitialRender.current = false;
      return () => {
        observer.disconnect();
      };
    }

    // Subsequent renders while open (when `content` changes).
    popupElement.style.setProperty('--popup-width', 'auto');
    popupElement.style.setProperty('--popup-height', 'auto');
    positionerElement.style.setProperty('--positioner-width', 'max-content');
    positionerElement.style.setProperty('--positioner-height', 'max-content');
    const newDimensions = (0, _getCssDimensions.getCssDimensions)(popupElement);
    popupElement.style.setProperty('--popup-width', `${previousDimensionsRef.current.width}px`);
    popupElement.style.setProperty('--popup-height', `${previousDimensionsRef.current.height}px`);
    restorePopupPosition();
    restorePopupTransform();
    restoreAvailableWidth();
    restoreAvailableHeight();
    onMeasureLayoutComplete?.(previousDimensionsRef.current, newDimensions);
    positionerElement.style.setProperty('--positioner-width', `${newDimensions.width}px`);
    positionerElement.style.setProperty('--positioner-height', `${newDimensions.height}px`);
    const abortController = new AbortController();
    animationFrame.request(() => {
      popupElement.style.setProperty('--popup-width', `${newDimensions.width}px`);
      popupElement.style.setProperty('--popup-height', `${newDimensions.height}px`);
      runOnceAnimationsFinish(() => {
        popupElement.style.setProperty('--popup-width', 'auto');
        popupElement.style.setProperty('--popup-height', 'auto');
      }, abortController.signal);
    });
    return () => {
      observer.disconnect();
      abortController.abort();
      animationFrame.cancel();
    };
  }, [content, popupElement, positionerElement, runOnceAnimationsFinish, animationFrame, enabled, mounted, onMeasureLayout, onMeasureLayoutComplete, anchoringStyles]);
}
function overrideElementStyle(element, property, value) {
  const originalValue = element.style.getPropertyValue(property);
  element.style.setProperty(property, value);
  return () => {
    element.style.setProperty(property, originalValue);
  };
}