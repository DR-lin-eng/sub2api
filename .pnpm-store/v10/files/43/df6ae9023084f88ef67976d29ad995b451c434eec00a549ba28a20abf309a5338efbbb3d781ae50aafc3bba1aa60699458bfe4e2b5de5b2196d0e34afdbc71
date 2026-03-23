import { createStaticStyles } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/MaskShadow/style.ts
const styles = createStaticStyles(({ css: css$1 }) => {
	return {
		bottomShadow: css$1`
      mask-image: linear-gradient(
        180deg,
        #000 calc(100% - var(--mask-shadow-size, 40%)),
        transparent
      );
    `,
		leftShadow: css$1`
      mask-image: linear-gradient(
        270deg,
        #000 calc(100% - var(--mask-shadow-size, 40%)),
        transparent
      );
    `,
		rightShadow: css$1`
      mask-image: linear-gradient(
        90deg,
        #000 calc(100% - var(--mask-shadow-size, 40%)),
        transparent
      );
    `,
		root: css$1`
      scrollbar-width: none;
      position: relative;
      overflow: hidden;

      -ms-overflow-style: none;

      &::-webkit-scrollbar {
        display: none;
      }
    `,
		topShadow: css$1`
      mask-image: linear-gradient(
        0deg,
        #000 calc(100% - var(--mask-shadow-size, 40%)),
        transparent
      );
    `
	};
});
const variants = cva(styles.root, {
	defaultVariants: { position: "bottom" },
	variants: { position: {
		top: styles.topShadow,
		bottom: styles.bottomShadow,
		left: styles.leftShadow,
		right: styles.rightShadow
	} }
});

//#endregion
export { variants };
//# sourceMappingURL=style.mjs.map