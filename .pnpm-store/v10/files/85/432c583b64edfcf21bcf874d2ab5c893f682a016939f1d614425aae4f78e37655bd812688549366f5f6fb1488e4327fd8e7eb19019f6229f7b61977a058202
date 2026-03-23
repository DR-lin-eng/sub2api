import { createStaticStyles } from "antd-style";

//#region src/awesome/Features/style.ts
const prefixCls = "ant";
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	const prefix = `${prefixCls}-features`;
	const coverCls = `${prefix}-cover`;
	const descCls = `${prefix}-description`;
	const titleCls = `${prefix}-title`;
	const imgCls = `${prefix}-img`;
	const scaleUnit = 20;
	const genSize = (size) => css$1`
    width: ${size}px;
    height: ${size}px;
    font-size: ${size * (22 / 24)}px;
  `;
	const withTransition = css$1`
    transition: all ${cssVar$1.motionDurationSlow} ${cssVar$1.motionEaseInOutCirc};
  `;
	return {
		cell: css$1`
      overflow: hidden;
    `,
		container: css$1`
      ${withTransition}
      position: relative;
      z-index: 1;

      overflow: hidden;

      height: 228px;
      max-height: 228px;
      padding: 24px;

      p {
        font-size: 16px;
        line-height: 1.2;
        text-align: start;
        word-break: break-word;
      }

      &:hover {
        .${coverCls} {
          width: 100%;
          height: calc(var(--features-row-num, 7) * ${scaleUnit}px);
          padding: 0;
          background: ${cssVar$1.colorFillContent};
        }

        .${imgCls} {
          ${genSize(100)};
        }

        .${descCls} {
          position: absolute;
          visibility: hidden;
          opacity: 0;
        }

        .${titleCls} {
          font-size: var(--features-title-hover-size, 20px);
        }
      }
    `,
		containerHasLink: css$1`
      ${withTransition}
      position: relative;
      z-index: 1;

      overflow: hidden;

      height: 228px;
      max-height: 228px;
      padding: 24px;

      p {
        font-size: 16px;
        line-height: 1.2;
        text-align: start;
        word-break: break-word;
      }

      &:hover {
        .${coverCls} {
          width: 100%;
          height: calc(var(--features-row-num, 7) * ${scaleUnit}px);
          padding: 0;
          background: ${cssVar$1.colorFillContent};
        }

        .${imgCls} {
          ${genSize(100)};
        }

        .${descCls} {
          position: absolute;
          visibility: hidden;
          opacity: 0;
        }

        .${titleCls} {
          font-size: 14px;
        }
      }
    `,
		desc: css$1`
      ${descCls}
      ${withTransition}
      pointer-events: none;
      color: ${cssVar$1.colorTextSecondary};

      quotient {
        position: relative;

        display: block;

        margin-block: 12px;
        margin-inline: 0;
        padding-inline-start: 12px;

        color: ${cssVar$1.colorTextDescription};
      }
    `,
		img: css$1`
      ${imgCls}
      ${withTransition}
      ${genSize(20)};
      color: ${cssVar$1.colorText};
    `,
		imgContainer: css$1`
      ${coverCls}
      ${withTransition}
      ${genSize(24)};
      padding: 4px;
      opacity: 0.8;
      border-radius: ${cssVar$1.borderRadius};
    `,
		link: css$1`
      ${withTransition};
      margin-block-start: 24px;
    `,
		title: css$1`
      ${titleCls}
      ${withTransition}
      pointer-events: none;

      margin-block: 16px;
      margin-inline: 0;

      font-size: 20px;
      line-height: ${cssVar$1.lineHeightHeading3};
      color: ${cssVar$1.colorText};
    `
	};
});

//#endregion
export { styles };
//# sourceMappingURL=style.mjs.map