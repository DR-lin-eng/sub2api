import { staticStylish } from "../../styles/theme/customStylishStatic.mjs";
import { createStaticStyles, cx } from "antd-style";

//#region src/awesome/BottomGradientButton/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => cx(staticStylish.resetLinkColor, css$1`
      overflow: hidden;
      font-weight: bold;
      color: ${cssVar$1.colorTextSecondary};
      transition: all 0.2s ease-in-out;

      &::before {
        content: '';

        position: absolute;
        inset-block-end: 0;

        display: block;

        width: 50%;
        height: 1px;

        opacity: 0;
        background-image: linear-gradient(to right, transparent, ${cssVar$1.gold}, transparent);

        transition: all 0.2s ease-in-out;
      }

      &:hover {
        &::before {
          opacity: 1;
        }
      }
    `));

//#endregion
export { styles };
//# sourceMappingURL=style.mjs.map