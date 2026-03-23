import { staticStylish } from "../../styles/theme/customStylishStatic.mjs";
import { createStaticStyles, cx } from "antd-style";

//#region src/awesome/GradientButton/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	const borderRadius = "var(--gradient-button-border-radius, var(--ant-border-radius))";
	return {
		buttonDark: css$1`
      position: relative;
      z-index: 1;
      border: none;
      border-radius: ${borderRadius} !important;

      &::before {
        ${staticStylish.gradientAnimation}
        content: '';

        position: absolute;
        z-index: -2;
        inset: 0;

        border-radius: ${borderRadius};
      }

      &::after {
        content: '';

        position: absolute;
        z-index: -1;
        inset-block-start: 1px;
        inset-inline-start: 1px;

        width: calc(100% - 2px);
        height: calc(100% - 2px);
        border-radius: calc(${borderRadius} - 1px);

        background: ${cssVar$1.colorBgLayout};
      }

      &:hover {
        &::after {
          background: color-mix(in srgb, ${cssVar$1.colorBgLayout} 90%, transparent);
        }
      }

      &:active {
        &::after {
          background: color-mix(in srgb, ${cssVar$1.colorBgLayout} 85%, transparent);
        }
      }
    `,
		buttonLight: css$1`
      position: relative;
      z-index: 1;
      border: none;
      border-radius: ${borderRadius} !important;

      &::before {
        ${staticStylish.gradientAnimation}
        content: '';

        position: absolute;
        z-index: -2;
        inset: 0;

        border-radius: ${borderRadius};
      }

      &::after {
        content: '';

        position: absolute;
        z-index: -1;
        inset-block-start: 1px;
        inset-inline-start: 1px;

        width: calc(100% - 2px);
        height: calc(100% - 2px);
        border-radius: calc(${borderRadius} - 1px);

        background: ${cssVar$1.colorBgContainer};
      }

      &:hover {
        &::after {
          background: color-mix(in srgb, ${cssVar$1.colorBgContainer} 95%, transparent);
        }
      }

      &:active {
        &::after {
          background: color-mix(in srgb, ${cssVar$1.colorBgContainer} 90%, transparent);
        }
      }
    `,
		glow: cx(staticStylish.gradientAnimation, css$1`
        position: absolute;
        z-index: -2;
        inset-block-start: 0;
        inset-inline-start: 0;

        width: 100%;
        height: 100%;
        border-radius: inherit;

        opacity: 0.5;
        filter: blur(0.5em);
      `)
	};
});

//#endregion
export { styles };
//# sourceMappingURL=style.mjs.map