import { createStaticStyles } from "antd-style";

//#region src/Markdown/markdown.style.ts
const IGNORE_CLASSNAME = ".ignore-markdown-style";
const styles = createStaticStyles(({ cssVar: cssVar$1, css: css$1 }) => {
	const __root = css$1`
    --lobe-markdown-font-size: 16px;
    --lobe-markdown-header-multiple: 1;
    --lobe-markdown-margin-multiple: 2;
    --lobe-markdown-line-height: 1.8;
    --lobe-markdown-border-radius: ${cssVar$1.borderRadiusLG};
    --lobe-markdown-border-color: ${cssVar$1.colorFillQuaternary};

    position: relative;

    width: 100%;
    max-width: 100%;
    padding-inline: 1px;

    font-size: var(--lobe-markdown-font-size);
    line-height: var(--lobe-markdown-line-height);
    word-break: break-word;
  `;
	const a = css$1`
    a {
      color: ${cssVar$1.colorInfoText};

      &:hover {
        color: ${cssVar$1.colorInfoHover};
      }
    }
  `;
	const blockquote = css$1`
    blockquote {
      margin-block: calc(var(--lobe-markdown-margin-multiple) * 0.5em);
      margin-inline: 0;
      padding-block: 0;
      padding-inline: 1em;
      border-inline-start: solid 4px ${cssVar$1.colorBorder};

      color: ${cssVar$1.colorTextSecondary};
    }
  `;
	const code = css$1`
    code {
      &:not(:has(span)) {
        display: inline;

        margin-inline: 0.25em;
        padding-block: 0.2em;
        padding-inline: 0.4em;
        border: 1px solid var(--lobe-markdown-border-color);
        border-radius: 0.25em;

        font-family: ${cssVar$1.fontFamilyCode};
        font-size: 0.875em;
        line-height: 1;
        word-break: break-word;
        white-space: break-spaces;

        background: ${cssVar$1.colorFillSecondary};
      }
    }
  `;
	const del = css$1`
    del {
      color: ${cssVar$1.colorTextDescription};
      text-decoration: line-through;
    }
  `;
	const details = css$1`
    details {
      margin-block: calc(var(--lobe-markdown-margin-multiple) * 0.5em);
      padding-block: 0.75em;
      padding-inline: 1em;
      border-radius: calc(var(--lobe-markdown-border-radius) * 1px);

      background: ${cssVar$1.colorFillTertiary};
      box-shadow: 0 0 0 1px var(--lobe-markdown-border-color);

      summary {
        cursor: pointer;
        display: flex;
        align-items: center;
        list-style: none;

        &::before {
          content: '';

          position: absolute;
          inset-inline-end: 1.25em;
          transform: rotateZ(-45deg);

          display: block;

          width: 0.4em;
          height: 0.4em;
          border-block-end: 1.5px solid ${cssVar$1.colorTextSecondary};
          border-inline-end: 1.5px solid ${cssVar$1.colorTextSecondary};

          font-family: ${cssVar$1.fontFamily};

          transition: transform 200ms ${cssVar$1.motionEaseOut};
        }
      }

      &[open] {
        summary {
          padding-block-end: 0.75em;
          border-block-end: 1px dashed ${cssVar$1.colorBorder};

          &::before {
            transform: rotateZ(45deg);
          }
        }
      }
    }
  `;
	const header = css$1`
    h1,
    h2,
    h3,
    h4,
    h5,
    h6 {
      margin-block: max(
        calc(var(--lobe-markdown-header-multiple) * var(--lobe-markdown-margin-multiple) * 0.4em),
        var(--lobe-markdown-font-size)
      );
      font-weight: bold;
      line-height: 1.25;
    }

    h1 {
      font-size: calc(
        var(--lobe-markdown-font-size) * (1 + 1.5 * var(--lobe-markdown-header-multiple))
      );
    }

    h2 {
      font-size: calc(var(--lobe-markdown-font-size) * (1 + var(--lobe-markdown-header-multiple)));
    }

    h3 {
      font-size: calc(
        var(--lobe-markdown-font-size) * (1 + 0.5 * var(--lobe-markdown-header-multiple))
      );
    }

    h4 {
      font-size: calc(
        var(--lobe-markdown-font-size) * (1 + 0.25 * var(--lobe-markdown-header-multiple))
      );
    }

    h5,
    h6 {
      font-size: calc(var(--lobe-markdown-font-size) * 1);
    }
  `;
	const hr = css$1`
    hr {
      width: 100%;
      margin-block: calc(var(--lobe-markdown-margin-multiple) * 1.5em);
      border-color: ${cssVar$1.colorBorder};
      border-style: dashed;
      border-width: 1px;
      border-block-start: none;
      border-inline-start: none;
      border-inline-end: none;
    }
  `;
	const img = css$1`
    img {
      max-width: 100%;
    }

    > img,
    > p > img {
      margin-block: calc(var(--lobe-markdown-margin-multiple) * 0.5em);
      border-radius: calc(var(--lobe-markdown-border-radius) * 1px);
      box-shadow: 0 0 0 1px var(--lobe-markdown-border-color);
    }
  `;
	const list = css$1`
    li {
      margin-block: calc(var(--lobe-markdown-margin-multiple) * 0.33em);

      p:first-child {
        display: inline;
      }
    }

    ul,
    ol {
      margin-block: calc(var(--lobe-markdown-margin-multiple) * 0.5em);
      margin-inline-start: 1em;
      padding-inline-start: 0;
      list-style-position: outside;

      > ul,
      > ol {
        margin-block: 0;
      }

      > li {
        margin-inline-start: 1em;
      }
    }

    ol {
      list-style: auto;
    }

    ul {
      list-style-type: none;

      > li {
        &::before {
          content: '-';

          position: absolute;

          display: inline-block;

          margin-inline: -1em 0.5em;

          opacity: 0.5;
        }
      }
    }

    .task-list-item {
      &::before {
        display: none !important;
      }

      input[type='checkbox'] {
        margin-block: 0 0.25em;
        margin-inline: -1.6em 0.2em;
        vertical-align: middle;
      }

      input[type='checkbox']:dir(rtl) {
        margin: 0 -1.6em 0.25em 0.2em;
      }
    }
  `;
	const p = css$1`
    p {
      margin-block: 4px;
      line-height: var(--lobe-markdown-line-height);
      letter-spacing: 0.02em;

      &:not(:first-child) {
        margin-block-start: calc(var(--lobe-markdown-margin-multiple) * 0.5em);
      }

      &:not(:last-child) {
        margin-block-end: calc(var(--lobe-markdown-margin-multiple) * 0.5em);
      }
    }
  `;
	const pre = css$1`
    pre {
      font-size: calc(var(--lobe-markdown-font-size) * 0.85);
    }
  `;
	const strong = css$1`
    strong {
      font-weight: 600;
    }
  `;
	const svg = css$1`
    svg {
      line-height: 1;
    }
  `;
	const table = css$1`
    table {
      unicode-bidi: isolate;
      overflow: auto hidden;
      display: block;
      border-spacing: 0;
      border-collapse: collapse;

      box-sizing: border-box;
      width: max-content;
      max-width: 100%;
      margin-block: calc(var(--lobe-markdown-margin-multiple) * 0.5em);
      border-radius: calc(var(--lobe-markdown-border-radius) * 1px);

      text-align: start;
      text-indent: initial;
      text-wrap: pretty;
      word-break: auto-phrase;
      overflow-wrap: break-word;

      background: ${cssVar$1.colorFillQuaternary};
      box-shadow: 0 0 0 1px ${cssVar$1.colorBorderSecondary};

      code {
        word-break: break-word;
      }

      thead {
        background: ${cssVar$1.colorFillQuaternary};
      }

      tr {
        box-shadow: 0 1px 0 ${cssVar$1.colorBorderSecondary};
      }

      th,
      td {
        min-width: 120px;
        padding-block: 0.75em;
        padding-inline: 1em;
        text-align: start;
      }
    }
  `;
	const video = css$1`
    > video,
    > p > video {
      margin-block: calc(var(--lobe-markdown-margin-multiple) * 0.5em);
      border-radius: calc(var(--lobe-markdown-border-radius) * 1px);
      box-shadow: 0 0 0 1px var(--lobe-markdown-border-color);
    }

    video {
      max-width: 100%;
    }
  `;
	const footnote = css$1`
    .footnotes {
      margin-block-start: calc(var(--lobe-markdown-margin-multiple) * 1em);
      font-size: smaller;
      color: #8b949e;

      #footnote-label {
        display: none;
      }

      > ol {
        margin: 0 !important;
      }
    }
  `;
	const sup = css$1`
    sup {
      position: relative;
      inset-block-start: -0.25em;

      font-size: 0.75em;
      line-height: var(--lobe-markdown-line-height);
      vertical-align: baseline;
    }
  `;
	return { root: css$1`
      :not(:has(${IGNORE_CLASSNAME})),
      .markdown {
        ${[
		__root,
		a,
		blockquote,
		code,
		del,
		details,
		header,
		hr,
		img,
		list,
		p,
		pre,
		strong,
		svg,
		table,
		video,
		footnote,
		css$1`
    sub {
      position: relative;
      inset-block-end: -0.25em;

      font-size: 0.75em;
      line-height: var(--lobe-markdown-line-height);
      vertical-align: baseline;
    }
  `,
		sup
	]}
      }
    ` };
});

//#endregion
export { styles };
//# sourceMappingURL=markdown.style.mjs.map