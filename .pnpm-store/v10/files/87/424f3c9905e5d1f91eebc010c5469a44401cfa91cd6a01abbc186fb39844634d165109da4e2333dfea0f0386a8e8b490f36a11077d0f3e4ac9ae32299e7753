import { createStaticStyles } from "antd-style";

//#region src/ColorSwatches/style.ts
const prefixCls = "ant";
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		active: css$1`
      box-shadow: inset 0 0 0 1px ${cssVar$1.colorFill};
    `,
		conic: css$1`
      background: conic-gradient(
        ${cssVar$1.red},
        ${cssVar$1.volcano},
        ${cssVar$1.orange},
        ${cssVar$1.gold},
        ${cssVar$1.yellow},
        ${cssVar$1.lime},
        ${cssVar$1.green},
        ${cssVar$1.cyan},
        ${cssVar$1.blue},
        ${cssVar$1.geekblue},
        ${cssVar$1.purple},
        ${cssVar$1.magenta},
        ${cssVar$1.red}
      );
      .${prefixCls}-color-picker-color-block {
        opacity: 0;
      }
    `,
		container: css$1`
      cursor: pointer;

      flex: none;

      width: var(--color-swatches-size, 24px);
      min-width: var(--color-swatches-size, 24px);
      height: var(--color-swatches-size, 24px);
      min-height: var(--color-swatches-size, 24px);

      background: ${cssVar$1.colorBgContainer};
      box-shadow: inset 0 0 0 1px ${cssVar$1.colorFillSecondary};

      &:hover {
        box-shadow:
          inset 0 0 0 1px rgba(0, 0, 0, 5%),
          0 0 0 2px ${cssVar$1.colorText};
      }
    `,
		picker: css$1`
      overflow: hidden;
      flex: none;

      width: var(--color-swatches-size, 24px);
      min-width: var(--color-swatches-size, 24px);
      height: var(--color-swatches-size, 24px);
      min-height: var(--color-swatches-size, 24px);
      padding: 0;
      border: none;

      box-shadow: inset 0 0 0 1px ${cssVar$1.colorFillSecondary};

      &:hover {
        box-shadow:
          inset 0 0 0 1px ${cssVar$1.colorFillSecondary},
          0 0 0 2px ${cssVar$1.colorText};
      }

      .${prefixCls}-color-picker-color-block {
        width: 100%;
        height: 100%;
        border: none;
        border-radius: inherit;
      }
    `,
		transparent: css$1`
      background-image: conic-gradient(
        ${cssVar$1.colorFillSecondary} 25%,
        transparent 25% 50%,
        ${cssVar$1.colorFillSecondary} 50% 75%,
        transparent 75% 100%
      );
      background-size: 50% 50%;
    `
	};
});

//#endregion
export { styles };
//# sourceMappingURL=style.mjs.map