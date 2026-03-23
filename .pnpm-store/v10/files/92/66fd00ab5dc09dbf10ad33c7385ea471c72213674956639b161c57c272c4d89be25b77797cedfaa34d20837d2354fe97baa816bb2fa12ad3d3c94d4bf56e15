import { createStaticStyles } from "antd-style";

//#region src/Grid/style.ts
const styles = createStaticStyles(({ css: css$1 }) => {
	return css$1`
    --rows: var(--grid-rows, 3);
    --max-item-width: var(--grid-max-item-width, 240px);
    --gap: var(--grid-gap, 1em);

    display: grid !important;
    grid-template-columns: repeat(
      auto-fill,
      minmax(
        max(var(--max-item-width), calc((100% - var(--gap) * (var(--rows) - 1)) / var(--rows))),
        1fr
      )
    );
  `;
});

//#endregion
export { styles };
//# sourceMappingURL=style.mjs.map