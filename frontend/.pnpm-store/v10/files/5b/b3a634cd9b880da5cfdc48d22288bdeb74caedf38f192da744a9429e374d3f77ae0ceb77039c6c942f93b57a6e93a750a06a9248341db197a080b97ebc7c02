import { createStaticStyles } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/Text/styles.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => ({
	code: css$1`
    font-family: ${cssVar$1.fontFamilyCode};
  `,
	danger: css$1`
    color: ${cssVar$1.colorError};
  `,
	delete: css$1`
    text-decoration: line-through;
  `,
	disabled: css$1`
    cursor: not-allowed;
    color: ${cssVar$1.colorTextDisabled};
  `,
	ellipsis: css$1`
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  `,
	ellipsisMulti: css$1`
    overflow: hidden;
    display: -webkit-box;
    -webkit-box-orient: vertical;
    text-overflow: ellipsis;
  `,
	h1: css$1`
    font-size: calc(${cssVar$1.fontSize} * 2.5);
    font-weight: bold;
    line-height: 1.25;
  `,
	h2: css$1`
    font-size: calc(${cssVar$1.fontSize} * 2);
    font-weight: bold;
    line-height: 1.25;
  `,
	h3: css$1`
    font-size: calc(${cssVar$1.fontSize} * 1.5);
    font-weight: bold;
    line-height: 1.25;
  `,
	h4: css$1`
    font-size: calc(${cssVar$1.fontSize} * 1.25);
    font-weight: bold;
    line-height: 1.25;
  `,
	h5: css$1`
    font-size: ${cssVar$1.fontSize};
    font-weight: bold;
    line-height: 1.25;
  `,
	info: css$1`
    color: ${cssVar$1.colorInfo};
  `,
	italic: css$1`
    font-style: italic;
  `,
	mark: css$1`
    color: #000;
    background-color: ${cssVar$1.yellow};
  `,
	p: css$1`
    margin-block: 0;
  `,
	secondary: css$1`
    color: ${cssVar$1.colorTextDescription};
  `,
	strong: css$1`
    font-weight: bold;
  `,
	success: css$1`
    color: ${cssVar$1.colorSuccess};
  `,
	text: css$1`
    color: ${cssVar$1.colorText};
  `,
	underline: css$1`
    text-decoration: underline;
  `,
	warning: css$1`
    color: ${cssVar$1.colorWarning};
  `
}));
const variants = cva(styles.text, {
	defaultVariants: {},
	variants: {
		as: {
			h1: styles.h1,
			h2: styles.h2,
			h3: styles.h3,
			h4: styles.h4,
			h5: styles.h5,
			p: styles.p
		},
		code: { true: styles.code },
		delete: { true: styles.delete },
		disabled: { true: styles.disabled },
		ellipsis: {
			multi: styles.ellipsisMulti,
			true: styles.ellipsis
		},
		italic: { true: styles.italic },
		mark: { true: styles.mark },
		strong: { true: styles.strong },
		type: {
			danger: styles.danger,
			info: styles.info,
			secondary: styles.secondary,
			success: styles.success,
			warning: styles.warning
		},
		underline: { true: styles.underline }
	}
});

//#endregion
export { variants };
//# sourceMappingURL=styles.mjs.map