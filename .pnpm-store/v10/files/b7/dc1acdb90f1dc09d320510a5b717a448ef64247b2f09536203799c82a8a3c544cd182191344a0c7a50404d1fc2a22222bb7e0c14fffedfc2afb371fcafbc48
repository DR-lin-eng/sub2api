import { keyframes } from "antd-style";

//#region src/styles/theme/customStylish.ts
const generateCustomStylish = ({ css: css$1, token, isDarkMode }) => {
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
	return {
		active: css$1`
      color: ${token.colorText};
      background: ${token.colorFillSecondary};

      &:hover {
        color: ${token.colorText};
        background: ${token.colorFill};
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
          background-color: ${token.colorFill};
          transition: background-color 500ms ${token.motionEaseOut};
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
        ${token.gold},
        ${token.magenta},
        ${token.geekblue},
        ${token.cyan}
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
      color: ${token.colorTextSecondary};

      &:hover {
        color: ${token.colorText};
      }
    `,
		shadow: css$1`
      box-shadow:
        0 1px 0 -1px ${isDarkMode ? token.colorBgLayout : token.colorBorder},
        0 1px 2px -0.5px ${isDarkMode ? token.colorBgLayout : token.colorBorder},
        0 2px 2px -1px ${isDarkMode ? token.colorBgLayout : token.colorBorderSecondary},
        0 3px 6px -4px ${isDarkMode ? token.colorBgLayout : token.colorBorderSecondary};
    `,
		variantBorderless: css$1`
      border: none;
      background: none;
      box-shadow: none;

      &:hover {
        background: ${token.colorFillTertiary};
      }
    `,
		variantBorderlessDanger: css$1`
      border: none;
      background: none;
      box-shadow: none;

      &:hover {
        background: ${token.colorErrorFillTertiary};
        box-shadow: inset 0 0 0 1px ${token.colorErrorFillTertiary};
      }
    `,
		variantBorderlessWithoutHover: css$1`
      border: none;
      background: none;
      box-shadow: none;
    `,
		variantFilled: css$1`
      background: ${token.colorFillTertiary};

      &:hover {
        background: ${token.colorFillSecondary};
      }
    `,
		variantFilledDanger: css$1`
      background: ${token.colorErrorFillTertiary};

      &:hover {
        background: ${token.colorErrorFillSecondary};
      }
    `,
		variantFilledWithoutHover: css$1`
      background: ${token.colorFillTertiary};
    `,
		variantOutlined: css$1`
      border: 1px solid ${token.colorBorderSecondary};
      background: ${token.colorBgContainer};

      &:hover {
        border: 1px solid ${token.colorBorder};
        background: ${token.colorBgContainer};
      }
    `,
		variantOutlinedDanger: css$1`
      border: 1px solid ${token.colorErrorBorder};

      &:hover {
        border: 1px solid ${token.colorErrorBorder};
      }
    `,
		variantOutlinedWithoutHover: css$1`
      border: 1px solid ${token.colorBorderSecondary};
      background: ${token.colorBgContainer};
    `
	};
};

//#endregion
export { generateCustomStylish };
//# sourceMappingURL=customStylish.mjs.map