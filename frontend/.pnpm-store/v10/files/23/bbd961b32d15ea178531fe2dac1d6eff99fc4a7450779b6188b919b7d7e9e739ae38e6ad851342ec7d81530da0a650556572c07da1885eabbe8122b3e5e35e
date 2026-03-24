import { createStaticStyles, keyframes } from "antd-style";

//#region src/styles/theme/customStylishStatic.ts
/**
* Static version of custom stylish utilities.
* This can be used with createStaticStyles for better performance.
*
* Note: Some styles that depend on isDarkMode or custom tokens may have limitations.
* For full dynamic support, use the regular customStylish from './customStylish'.
*/
const gradient = keyframes`
  0% {
    background-position: 0% 50%;
  }
  50% {
    background-position: 100% 50%;
  }
  100% {
    background-position: 0% 50%;
  }
`;
const staticStylish = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => ({
	active: css$1`
    color: ${cssVar$1.colorText};
    background: ${cssVar$1.colorFillSecondary};

    &:hover {
      color: ${cssVar$1.colorText};
      background: ${cssVar$1.colorFill};
    }
  `,
	blur: css$1`
    backdrop-filter: saturate(150%) blur(10px);
  `,
	blurStrong: css$1`
    backdrop-filter: saturate(150%) blur(36px);
  `,
	bottomScrollbar: css$1`
    ::-webkit-scrollbar {
      width: 0;
      height: 4px;
      background-color: transparent;

      &-thumb {
        border-radius: 4px;
        background-color: ${cssVar$1.colorFill};
        transition: background-color 500ms ${cssVar$1.motionEaseOut};
      }

      &-corner {
        display: none;
        width: 0;
        height: 0;
      }
    }
  `,
	disabled: css$1`
    cursor: not-allowed;
    opacity: 0.5;
  `,
	gradientAnimation: css$1`
    border-radius: inherit;
    background-image: linear-gradient(
      -45deg,
      ${cssVar$1.gold},
      ${cssVar$1.magenta},
      ${cssVar$1.geekblue},
      ${cssVar$1.cyan}
    );
    background-size: 400% 400%;
    animation: 5s ${gradient} 5s ease infinite;
  `,
	noScrollbar: css$1`
    ::-webkit-scrollbar {
      display: none;
      width: 0;
      height: 0;
      background-color: transparent;
    }
  `,
	resetLinkColor: css$1`
    cursor: pointer;
    color: ${cssVar$1.colorTextSecondary};

    &:hover {
      color: ${cssVar$1.colorText};
    }
  `,
	shadow: css$1`
    box-shadow:
      0 1px 0 -1px ${cssVar$1.colorBorder},
      0 1px 2px -0.5px ${cssVar$1.colorBorder},
      0 2px 2px -1px ${cssVar$1.colorBorderSecondary},
      0 3px 6px -4px ${cssVar$1.colorBorderSecondary};
  `,
	variantBorderless: css$1`
    border: none;
    background: none;
    box-shadow: none;

    &:hover {
      background: ${cssVar$1.colorFillTertiary};
    }
  `,
	variantBorderlessDanger: css$1`
    border: none;
    background: none;
    box-shadow: none;

    &:hover {
      background: ${cssVar$1.colorErrorBg};
      box-shadow: inset 0 0 0 1px ${cssVar$1.colorErrorBg};
    }
  `,
	variantBorderlessWithoutHover: css$1`
    border: none;
    background: none;
    box-shadow: none;
  `,
	variantFilled: css$1`
    background: ${cssVar$1.colorFillTertiary};

    &:hover {
      background: ${cssVar$1.colorFillSecondary};
    }
  `,
	variantFilledDanger: css$1`
    background: ${cssVar$1.colorErrorBg};

    &:hover {
      background: ${cssVar$1.colorErrorBgHover};
    }
  `,
	variantFilledWithoutHover: css$1`
    background: ${cssVar$1.colorFillTertiary};
  `,
	variantOutlined: css$1`
    border: 1px solid ${cssVar$1.colorBorderSecondary};
    background: ${cssVar$1.colorBgContainer};

    &:hover {
      border: 1px solid ${cssVar$1.colorBorder};
      background: ${cssVar$1.colorBgContainer};
    }
  `,
	variantOutlinedDanger: css$1`
    border: 1px solid ${cssVar$1.colorErrorBorder};

    &:hover {
      border: 1px solid ${cssVar$1.colorErrorBorder};
    }
  `,
	variantOutlinedWithoutHover: css$1`
    border: 1px solid ${cssVar$1.colorBorderSecondary};
    background: ${cssVar$1.colorBgContainer};
  `
}));

//#endregion
export { staticStylish };
//# sourceMappingURL=customStylishStatic.mjs.map