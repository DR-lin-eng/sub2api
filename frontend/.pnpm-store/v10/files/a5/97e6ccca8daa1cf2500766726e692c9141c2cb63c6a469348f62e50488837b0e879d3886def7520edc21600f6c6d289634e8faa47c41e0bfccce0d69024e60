import { useEffect, useState } from "react";

//#region src/ScrollShadow/useScrollOverflow.ts
const useScrollOverflow = ({ domRef, offset = 0, orientation = "vertical", isEnabled = true, onVisibilityChange, updateDeps = [] }) => {
	const [scrollState, setScrollState] = useState({
		bottom: false,
		left: false,
		right: false,
		top: false
	});
	useEffect(() => {
		const element = domRef.current;
		if (!element || !isEnabled) return;
		const checkScroll = () => {
			const newState = { ...scrollState };
			if (orientation === "vertical") if (element.scrollHeight > element.clientHeight) {
				newState.top = element.scrollTop > offset;
				newState.bottom = element.scrollTop + element.clientHeight < element.scrollHeight - offset;
			} else {
				newState.top = false;
				newState.bottom = false;
			}
			else if (element.scrollWidth > element.clientWidth) {
				newState.left = element.scrollLeft > offset;
				newState.right = element.scrollLeft + element.clientWidth < element.scrollWidth - offset;
			} else {
				newState.left = false;
				newState.right = false;
			}
			setScrollState(newState);
			onVisibilityChange?.(newState);
		};
		checkScroll();
		element.addEventListener("scroll", checkScroll);
		window.addEventListener("resize", checkScroll);
		const resizeObserver = new ResizeObserver(checkScroll);
		resizeObserver.observe(element);
		return () => {
			element.removeEventListener("scroll", checkScroll);
			window.removeEventListener("resize", checkScroll);
			resizeObserver.disconnect();
		};
	}, [
		domRef,
		offset,
		orientation,
		isEnabled,
		onVisibilityChange,
		...updateDeps
	]);
	return scrollState;
};

//#endregion
export { useScrollOverflow };
//# sourceMappingURL=useScrollOverflow.mjs.map