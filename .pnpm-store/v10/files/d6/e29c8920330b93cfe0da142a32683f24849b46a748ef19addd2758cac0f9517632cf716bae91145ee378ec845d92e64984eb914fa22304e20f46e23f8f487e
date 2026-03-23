import { createStaticStyles } from "antd-style";

//#region src/awesome/Spotlight/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => ({
	spotlightDark: css$1`
    pointer-events: none;

    position: absolute;
    z-index: 1;
    inset: 0;

    border-radius: inherit;

    opacity: var(--spotlight-opacity, 0.1);
    background: radial-gradient(
      var(--spotlight-size, 64px) circle at var(--spotlight-x, 0) var(--spotlight-y, 0),
      ${cssVar$1.colorText},
      transparent
    );

    transition: all 0.2s;
  `,
	spotlightDarkOutside: css$1`
    pointer-events: none;

    position: absolute;
    z-index: 1;
    inset: 0;

    border-radius: inherit;

    opacity: 0;
    background: radial-gradient(
      var(--spotlight-size, 64px) circle at var(--spotlight-x, 0) var(--spotlight-y, 0),
      ${cssVar$1.colorText},
      transparent
    );

    transition: all 0.2s;
  `,
	spotlightLight: css$1`
    pointer-events: none;

    position: absolute;
    z-index: 1;
    inset: 0;

    border-radius: inherit;

    opacity: var(--spotlight-opacity, 0.1);
    background: radial-gradient(
      var(--spotlight-size, 64px) circle at var(--spotlight-x, 0) var(--spotlight-y, 0),
      #fff,
      ${cssVar$1.colorTextQuaternary}
    );

    transition: all 0.2s;
  `,
	spotlightLightOutside: css$1`
    pointer-events: none;

    position: absolute;
    z-index: 1;
    inset: 0;

    border-radius: inherit;

    opacity: 0;
    background: radial-gradient(
      var(--spotlight-size, 64px) circle at var(--spotlight-x, 0) var(--spotlight-y, 0),
      #fff,
      ${cssVar$1.colorTextQuaternary}
    );

    transition: all 0.2s;
  `
}));

//#endregion
export { styles };
//# sourceMappingURL=style.mjs.map