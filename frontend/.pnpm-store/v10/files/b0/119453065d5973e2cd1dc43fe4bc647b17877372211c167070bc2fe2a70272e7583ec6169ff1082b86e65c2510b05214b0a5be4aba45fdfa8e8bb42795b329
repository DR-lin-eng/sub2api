import { createStaticStyles, keyframes } from "antd-style";

//#region src/Skeleton/style.ts
const shimmer = keyframes`
  0% {
    opacity: 1;
  }
  50% {
    opacity: .5;
  }
  100% {
    opacity: 1;
  }
`;
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		active: css$1`
      background: ${cssVar$1.colorFillSecondary};
      animation: ${shimmer} 2s linear infinite;
    `,
		avatar: css$1`
      flex-shrink: 0;
    `,
		base: css$1`
      user-select: none;

      position: relative;

      overflow: hidden;

      border-radius: ${cssVar$1.borderRadius};

      background: ${cssVar$1.colorFillTertiary};
    `,
		text: css$1`
      display: flex;
      flex: 1;
      flex-direction: column;
      gap: ${cssVar$1.paddingXS};

      width: 100%;
    `
	};
});

//#endregion
export { styles };
//# sourceMappingURL=style.mjs.map