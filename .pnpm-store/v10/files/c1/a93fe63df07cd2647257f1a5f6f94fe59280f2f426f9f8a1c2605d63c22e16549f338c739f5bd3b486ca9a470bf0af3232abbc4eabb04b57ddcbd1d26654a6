"use strict";
'use client';

var _interopRequireWildcard = require("@babel/runtime/helpers/interopRequireWildcard").default;
Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.useAnimationsFinished = useAnimationsFinished;
var ReactDOM = _interopRequireWildcard(require("react-dom"));
var _useAnimationFrame = require("@base-ui/utils/useAnimationFrame");
var _useStableCallback = require("@base-ui/utils/useStableCallback");
var _resolveRef = require("./resolveRef");
/**
 * Executes a function once all animations have finished on the provided element.
 * @param elementOrRef - The element to watch for animations.
 * @param waitForNextTick - Whether to wait for the next tick before checking for animations.
 * @param treatAbortedAsFinished - Whether to treat aborted animations as finished. If `false`, and there are aborted animations,
 *   the function will check again if any new animations have started and wait for them to finish.
 * @returns A function that takes a callback to execute once all animations have finished, and an optional AbortSignal to abort the callback
 */
function useAnimationsFinished(elementOrRef, waitForNextTick = false, treatAbortedAsFinished = true) {
  const frame = (0, _useAnimationFrame.useAnimationFrame)();
  return (0, _useStableCallback.useStableCallback)((fnToExecute,
  /**
   * An optional [AbortSignal](https://developer.mozilla.org/en-US/docs/Web/API/AbortSignal) that
   * can be used to abort `fnToExecute` before all the animations have finished.
   * @default null
   */
  signal = null) => {
    frame.cancel();
    const element = (0, _resolveRef.resolveRef)(elementOrRef);
    if (element == null) {
      return;
    }
    if (typeof element.getAnimations !== 'function' || globalThis.BASE_UI_ANIMATIONS_DISABLED) {
      fnToExecute();
    } else {
      frame.request(() => {
        function exec() {
          if (!element) {
            return;
          }
          Promise.all(element.getAnimations().map(anim => anim.finished)).then(() => {
            if (signal != null && signal.aborted) {
              return;
            }

            // Synchronously flush the unmounting of the component so that the browser doesn't
            // paint: https://github.com/mui/base-ui/issues/979
            ReactDOM.flushSync(fnToExecute);
          }).catch(() => {
            if (treatAbortedAsFinished) {
              if (signal != null && signal.aborted) {
                return;
              }
              ReactDOM.flushSync(fnToExecute);
            } else if (element.getAnimations().length > 0 && element.getAnimations().some(anim => anim.pending || anim.playState !== 'finished')) {
              // Sometimes animations can be aborted because a property they depend on changes while the animation plays.
              // In such cases, we need to re-check if any new animations have started.
              exec();
            }
          });
        }

        // `open: true` animations need to wait for the next tick to be detected
        if (waitForNextTick) {
          frame.request(exec);
        } else {
          exec();
        }
      });
    }
  });
}