'use client';

import { jsx } from "react/jsx-runtime";
import { Divider } from "antd";
import { createStaticStyles, cx } from "antd-style";

//#region src/Form/components/FormDivider.tsx
const styles = createStaticStyles(({ css: css$1 }) => {
	return { root: css$1`
      margin: 0;
      opacity: 0.66;
    ` };
});
const FormDivider = ({ visible = true, style, className, ...rest }) => {
	return /* @__PURE__ */ jsx(Divider, {
		className: cx(styles.root, className),
		style: {
			opacity: visible ? 1 : 0,
			...style
		},
		...rest
	});
};
FormDivider.displayName = "FormDivider";
var FormDivider_default = FormDivider;

//#endregion
export { FormDivider_default as default };
//# sourceMappingURL=FormDivider.mjs.map