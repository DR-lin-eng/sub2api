import { createStaticStyles } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/Markdown/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		chat: css$1`
      ol,
      ul {
        > li {
          &::marker {
            color: ${cssVar$1.colorTextSecondary} !important;
          }

          > li {
            &::marker {
              color: ${cssVar$1.colorTextSecondary} !important;
            }
          }
        }
      }

      ul {
        list-style: unset;

        > li {
          &::before {
            content: unset;
            display: unset;
          }
        }
      }
    `,
		gfm: css$1`
      .markdown-alert {
        margin-block: calc(var(--lobe-markdown-margin-multiple) * 0.5em);
        padding-inline-start: 1em;
        border-inline-start: solid 4px ${cssVar$1.colorBorder};

        > p {
          margin-block-start: 0 !important;
        }
      }

      .markdown-alert > :first-child {
        margin-block-start: 0;
      }

      .markdown-alert > :last-child {
        margin-block-end: 0;
      }

      .markdown-alert-note {
        border-inline-start-color: ${cssVar$1.colorInfo};
      }

      .markdown-alert-tip {
        border-inline-start-color: ${cssVar$1.colorSuccess};
      }

      .markdown-alert-important {
        border-inline-start-color: ${cssVar$1.purple};
      }

      .markdown-alert-warning {
        border-inline-start-color: ${cssVar$1.colorWarning};
      }

      .markdown-alert-caution {
        border-inline-start-color: ${cssVar$1.colorError};
      }

      .markdown-alert-title {
        display: flex;
        align-items: center;
        margin-block-end: 0.5em !important;
        font-weight: 500;
      }

      .markdown-alert-note .markdown-alert-title {
        color: ${cssVar$1.colorInfo};
        fill: ${cssVar$1.colorInfo};
      }

      .markdown-alert-tip .markdown-alert-title {
        color: ${cssVar$1.colorSuccess};
        fill: ${cssVar$1.colorSuccess};
      }

      .markdown-alert-important .markdown-alert-title {
        color: ${cssVar$1.purple};
        fill: ${cssVar$1.purple};
      }

      .markdown-alert-warning .markdown-alert-title {
        color: ${cssVar$1.colorWarning};
        fill: ${cssVar$1.colorWarning};
      }

      .markdown-alert-caution .markdown-alert-title {
        color: ${cssVar$1.colorError};
        fill: ${cssVar$1.colorError};
      }

      /* Style the footnotes section. */

      .octicon {
        overflow: visible !important;
        display: inline-block;
        margin-inline-end: 0.5em;
        vertical-align: text-bottom;
      }

      .sr-only {
        position: absolute;

        overflow: hidden;

        width: 1px;
        height: 1px;
        padding: 0;
        border: 0;

        word-wrap: normal;

        clip: rect(0, 0, 0, 0);
      }

      sup:has(*[aria-describedby='footnote-label']) {
        margin-inline: 2px;
        vertical-align: super !important;

        [data-footnote-ref] {
          display: inline-block;

          width: 16px;
          height: 16px;
          border-radius: 4px;

          font-family: ${cssVar$1.fontFamilyCode};
          font-size: 10px;
          color: ${cssVar$1.colorTextSecondary} !important;
          text-align: center;

          background: ${cssVar$1.colorFillSecondary};
        }
      }

      code.color-preview {
        position: relative;
        display: inline-flex !important;
        gap: 0.4em;

        &::after {
          content: '';

          width: 0.66em;
          height: 0.66em;
          border: 1px solid ${cssVar$1.colorFill};
          border-radius: 50%;

          background-color: attr(data-color);

          /* Fallback for browsers that don't support attr() in background */
          background-color: var(--color-preview-color, #000);
        }
      }
    `,
		latex: css$1`
      .katex-error {
        color: ${cssVar$1.colorTextDescription} !important;
      }

      .katex-html {
        overflow: auto hidden;
        padding: 3px;

        .base {
          margin-block: 0;
          margin-inline: auto;
        }

        .tag {
          position: relative !important;
          display: inline-block;
          padding-inline-start: 0.5rem;
        }
      }
    `,
		root: css$1`
      position: relative;
      overflow: hidden;
      max-width: 100%;
    `
	};
});
const variants = cva(styles.root, {
	defaultVariants: {
		enableGfm: true,
		enableLatex: true,
		variant: "default"
	},
	variants: {
		variant: {
			default: null,
			chat: styles.chat
		},
		enableLatex: {
			true: styles.latex,
			false: null
		},
		enableGfm: {
			true: styles.gfm,
			false: null
		}
	}
});

//#endregion
export { styles, variants };
//# sourceMappingURL=style.mjs.map