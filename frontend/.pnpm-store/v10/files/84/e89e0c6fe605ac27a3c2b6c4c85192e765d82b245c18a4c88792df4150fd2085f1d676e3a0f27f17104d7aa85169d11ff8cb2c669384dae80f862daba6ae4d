import { createStaticStyles } from "antd-style";

//#region src/storybook/StoryBook/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1, responsive: responsive$1 }) => {
	return {
		editor: css$1`
      width: inherit;
      min-height: inherit;
    `,
		left: css$1`
      position: relative;
      overflow: auto;
    `,
		leftWithPadding: css$1`
      position: relative;
      overflow: auto;
      padding-block: 40px;
      padding-inline: 24px;
    `,
		leva: css$1`
      --leva-sizes-controlWidth: 66%;
      --leva-colors-elevation1: ${cssVar$1.colorFillSecondary};
      --leva-colors-elevation2: transparent;
      --leva-colors-elevation3: ${cssVar$1.colorFillSecondary};
      --leva-colors-accent1: ${cssVar$1.colorPrimary};
      --leva-colors-accent2: ${cssVar$1.colorPrimaryHover};
      --leva-colors-accent3: ${cssVar$1.colorPrimaryActive};
      --leva-colors-highlight1: ${cssVar$1.colorTextTertiary};
      --leva-colors-highlight2: ${cssVar$1.colorTextSecondary};
      --leva-colors-highlight3: ${cssVar$1.colorText};
      --leva-colors-vivid1: ${cssVar$1.colorWarning};
      --leva-shadows-level1: unset;
      --leva-shadows-level2: unset;
      --leva-fonts-mono: ${cssVar$1.fontFamilyCode};

      overflow: auto;

      width: 100%;
      height: 100%;
      padding-block: 6px;
      padding-inline: 0;

      > div {
        background: transparent;

        > div {
          background: transparent;
        }
      }

      input:checked + label > svg {
        stroke: ${cssVar$1.colorBgLayout};
      }

      button {
        --leva-colors-accent2: ${cssVar$1.colorFillSecondary};
      }
    `,
		right: css$1`
      background: ${cssVar$1.colorBgLayout};

      ${responsive$1.sm} {
        .draggable-panel-fixed {
          width: 100% !important;
        }
      }
    `
	};
});

//#endregion
export { styles };
//# sourceMappingURL=style.mjs.map