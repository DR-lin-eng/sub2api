import { createStaticStyles } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/ScrollShadow/style.ts
const styles = createStaticStyles(({ css: css$1 }) => {
	return {
		bottomShadow: css$1`
      mask-image: linear-gradient(
        180deg,
        #000 calc(100% - var(--scroll-shadow-size, 40%)),
        transparent
      );
    `,
		hideScrollBar: css$1`
      scrollbar-width: none;

      -ms-overflow-style: none;

      &::-webkit-scrollbar {
        display: none;
      }
    `,
		horizontal: css$1`
      overflow-x: auto;
    `,
		leftRightShadow: css$1`
      mask-image: linear-gradient(
        to right,
        #000,
        #000,
        transparent 0,
        #000 var(--scroll-shadow-size, 40%),
        #000 calc(100% - var(--scroll-shadow-size, 40%)),
        transparent
      );
    `,
		leftShadow: css$1`
      mask-image: linear-gradient(
        270deg,
        #000 calc(100% - var(--scroll-shadow-size, 40%)),
        transparent
      );
    `,
		rightShadow: css$1`
      mask-image: linear-gradient(
        90deg,
        #000 calc(100% - var(--scroll-shadow-size, 40%)),
        transparent
      );
    `,
		root: css$1`
      position: relative;
      overflow: hidden;
    `,
		topBottomShadow: css$1`
      mask-image: linear-gradient(
        #000,
        #000,
        transparent 0,
        #000 var(--scroll-shadow-size, 40%),
        #000 calc(100% - var(--scroll-shadow-size, 40%)),
        transparent
      );
    `,
		topShadow: css$1`
      mask-image: linear-gradient(
        0deg,
        #000 calc(100% - var(--scroll-shadow-size, 40%)),
        transparent
      );
    `,
		vertical: css$1`
      overflow-y: auto;
    `
	};
});
const variants = cva(styles.root, {
	defaultVariants: {
		hideScrollBar: false,
		orientation: "vertical",
		scrollPosition: "none"
	},
	variants: {
		orientation: {
			horizontal: styles.horizontal,
			vertical: styles.vertical
		},
		hideScrollBar: {
			true: styles.hideScrollBar,
			false: null
		},
		scrollPosition: {
			"none": null,
			"top": styles.topShadow,
			"bottom": styles.bottomShadow,
			"top-bottom": styles.topBottomShadow,
			"left": styles.leftShadow,
			"right": styles.rightShadow,
			"left-right": styles.leftRightShadow
		}
	}
});

//#endregion
export { variants };
//# sourceMappingURL=style.mjs.map