import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles, cx } from "antd-style";

//#region src/ImageSelect/styles.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		active: css$1`
      color: ${cssVar$1.colorText};
    `,
		container: css$1`
      cursor: pointer;
      color: ${cssVar$1.colorTextDescription};
    `,
		img: cx(staticStylish.variantFilled, css$1`
        border-radius: ${cssVar$1.borderRadius};

        &:hover {
          box-shadow: 0 0 0 2px ${cssVar$1.colorText};
        }
      `),
		imgActive: cx(staticStylish.active, css$1`
        box-shadow: 0 0 0 2px ${cssVar$1.colorTextTertiary};
      `)
	};
});

//#endregion
export { styles };
//# sourceMappingURL=styles.mjs.map