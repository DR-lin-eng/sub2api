import { createStaticStyles } from "antd-style";

//#region src/DraggableSideNav/style.ts
const LAYOUT = {
	offset: 16,
	toggleLength: 54,
	toggleShort: 16
};
const prefixCls = "ant";
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => ({
	body: css$1`
    /* Smooth scroll behavior */
    scroll-behavior: smooth;
    overflow: hidden auto;
    flex: 1;

    /* Better scrollbar styling */
    &::-webkit-scrollbar {
      width: 6px;
    }

    &::-webkit-scrollbar-track {
      background: transparent;
    }

    &::-webkit-scrollbar-thumb {
      border-radius: 3px;
      background: ${cssVar$1.colorBorderSecondary};

      &:hover {
        background: ${cssVar$1.colorBorder};
      }
    }
  `,
	container: css$1`
    /* Width transition controlled by inline style */

    /* Ensure smooth animations */
    will-change: width;

    position: relative;

    display: flex;
    flex-direction: column;

    height: 100%;
  `,
	contentContainer: css$1`
    overflow: hidden;
    display: flex;
    flex-direction: column;

    height: 100%;
    border-inline-end: 1px solid ${cssVar$1.colorBorderSecondary};

    background: var(--draggable-side-nav-bg, ${cssVar$1.colorBgLayout});
  `,
	contentContainerNoBorder: css$1`
    overflow: hidden;
    display: flex;
    flex-direction: column;

    height: 100%;
    border-inline-end: 0 solid ${cssVar$1.colorBorderSecondary};

    background: var(--draggable-side-nav-bg, ${cssVar$1.colorBgLayout});
  `,
	footer: css$1`
    flex-shrink: 0;
  `,
	handlerIcon: css$1`
    /* Icon transitions are now handled by motion */
    display: flex;
    align-items: center;
    justify-content: center;
  `,
	header: css$1`
    flex-shrink: 0;
  `,
	menuOverride: css$1`
    .${prefixCls}-menu {
      .${prefixCls}-menu-item {
        display: flex;
        gap: 8px;
        align-items: center;
        justify-content: center;

        height: unset;
        min-height: 36px;
        padding-block: 4px;
        padding-inline: 8px !important;
      }

      .${prefixCls}-menu-item-group-title {
        overflow: hidden;
        padding-inline: 8px;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .${prefixCls}-menu-item-icon {
        position: absolute;
        inset-inline-start: 0;

        display: flex !important;
        flex: none;
        align-items: center;
        justify-content: center;

        width: 36px;
        height: 36px;
      }

      .${prefixCls}-menu-title-content {
        overflow: hidden;
        flex: 1;

        margin: 0 !important;
        padding-inline-start: 36px;

        line-height: 1.5;
      }

      &.${prefixCls}-menu-inline-collapsed {
        .ant-menu-title-content {
          display: none;
          width: 0;
          opacity: 0;
        }

        .${prefixCls}-menu-item {
          display: flex;
          align-items: center;
          justify-content: center;

          width: 36px !important;
          height: 36px !important;
        }
      }
    }
  `,
	resizeHandle: css$1`
    cursor: col-resize;

    position: absolute;
    inset-block: 0 0;

    width: 8px;

    transition: background-color 0.2s ease;

    &::after {
      content: '';

      position: absolute;
      inset-block: 0;
      inset-inline-start: 50%;
      transform: translateX(-50%);

      width: 2px;

      background: transparent;

      transition: all 0.25s cubic-bezier(0.22, 1, 0.36, 1);
    }
  `,
	resizeHandleHighlight: css$1`
    &:hover {
      &::after {
        width: 3px;
        background: ${cssVar$1.colorPrimary};
        box-shadow: 0 0 8px color-mix(in srgb, ${cssVar$1.colorPrimary} 25%, transparent);
      }
    }

    &:active {
      &::after {
        background: ${cssVar$1.colorPrimaryActive};
      }
    }
  `,
	resizeHandleLeft: css$1`
    inset-inline-end: -4px;
  `,
	resizeHandleRight: css$1`
    inset-inline-start: -4px;
  `,
	toggleLeft: css$1`
    inset-inline-end: -${LAYOUT.offset}px;
    width: ${LAYOUT.toggleShort}px;
    height: 100%;

    > div {
      inset-block-start: 50%;

      width: ${LAYOUT.toggleShort}px;
      height: ${LAYOUT.toggleLength}px;
      margin-block-start: -${LAYOUT.toggleLength / 2}px;
      margin-inline-start: -1px;
      border-inline-start-width: 0;
      border-radius: 0 ${cssVar$1.borderRadiusLG} ${cssVar$1.borderRadiusLG} 0;
    }
  `,
	toggleRight: css$1`
    inset-inline-start: -${LAYOUT.offset}px;
    width: ${LAYOUT.toggleShort}px;
    height: 100%;

    > div {
      inset-block-start: 50%;

      width: ${LAYOUT.toggleShort}px;
      height: ${LAYOUT.toggleLength}px;
      margin-block-start: -${LAYOUT.toggleLength / 2}px;
      margin-inline-end: -1px;
      border-inline-end-width: 0;
      border-radius: ${cssVar$1.borderRadiusLG} 0 0 ${cssVar$1.borderRadiusLG}; /* 右侧面板，handle 在左边，左侧圆角 */
    }
  `,
	toggleRoot: css$1`
    pointer-events: none;
    position: absolute;
    z-index: 50;

    /* Smooth transitions for all states */
    transition: opacity 0.25s cubic-bezier(0.22, 1, 0.36, 1);

    &:has(> div) {
      pointer-events: all;
    }

    > div {
      pointer-events: all;
      cursor: pointer;

      position: absolute;

      border: 1px solid ${cssVar$1.colorBorder};

      color: ${cssVar$1.colorTextTertiary};

      background: var(--draggable-side-nav-bg, ${cssVar$1.colorBgLayout});
      backdrop-filter: blur(8px);

      /* Enhanced transitions with backdrop blur */
      transition:
        color 0.2s ${cssVar$1.motionEaseOut},
        transform 0.2s ${cssVar$1.motionEaseOut},
        box-shadow 0.2s ${cssVar$1.motionEaseOut};

      &:hover {
        color: ${cssVar$1.colorTextSecondary};
      }

      &:active {
        transform: scale(0.95);
        color: ${cssVar$1.colorText};
      }
    }
  `
}));

//#endregion
export { styles };
//# sourceMappingURL=style.mjs.map