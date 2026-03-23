import { createStaticStyles, responsive } from "antd-style";

//#region src/chat/ChatItem/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	const blockStylish = css$1`
    padding-block: 8px;
    padding-inline: 12px;
    border: 1px solid color-mix(in srgb, ${cssVar$1.colorBorderSecondary} 66%, transparent);
    border-radius: ${cssVar$1.borderRadiusLG};

    background-color: ${cssVar$1.colorBgContainer};
  `;
	const rawStylishWithTitle = css$1`
    padding-block-start: 0;
  `;
	const rawStylishWithoutTitle = css$1`
    padding-block-start: 6px;
  `;
	const rawContainerStylish = css$1`
    margin-block-end: -16px;
    transition: background-color 100ms ${cssVar$1.motionEaseOut};
  `;
	const editingStylish = css$1`
    width: 100%;
  `;
	return {
		actionsBubbleLeft: css$1`
      flex: none;
      align-self: flex-end;
      justify-content: flex-end;
    `,
		actionsBubbleRight: css$1`
      flex: none;
      align-self: flex-end;
      justify-content: flex-start;
    `,
		actionsDocsLeft: css$1`
      flex: none;
      align-self: flex-start;
      justify-content: flex-end;
    `,
		actionsDocsRight: css$1`
      flex: none;
      align-self: flex-end;
      justify-content: flex-start;
    `,
		actionsEditing: css$1`
      pointer-events: none !important;
      opacity: 0 !important;
    `,
		avatarContainer: css$1`
      position: relative;
      flex: none;
      width: var(--chat-item-avatar-size, 40px);
      height: var(--chat-item-avatar-size, 40px);
    `,
		avatarGroupContainer: css$1`
      width: var(--chat-item-avatar-size, 40px);
    `,
		container: css$1`
      position: relative;

      width: 100%;
      max-width: 100vw;
      padding-block: 24px 12px;
      padding-inline: 12px;

      time {
        display: inline-block;
        white-space: nowrap;
      }

      div[role='menubar'] {
        display: flex;
      }

      time,
      div[role='menubar'] {
        pointer-events: none;
        opacity: 0;
        transition: opacity 200ms ${cssVar$1.motionEaseOut};
      }

      &:hover {
        time,
        div[role='menubar'] {
          pointer-events: unset;
          opacity: 1;
        }
      }

      div[role='menubar']:has([data-popup-open]),
      div[role='menubar'][data-popup-open] {
        pointer-events: unset !important;
        opacity: 1 !important;

        [data-popup-open] {
          background: ${cssVar$1.colorFillTertiary};
        }
      }

      ${responsive.sm} {
        padding-block-start: 12px;
        padding-inline: 8px;
      }
    `,
		containerDocs: css$1`
      ${rawContainerStylish}
      position: relative;

      width: 100%;
      max-width: 100vw;
      padding-block: 24px 12px;
      padding-inline: 12px;

      time {
        display: inline-block;
        white-space: nowrap;
      }

      div[role='menubar'] {
        display: flex;
      }

      time,
      div[role='menubar'] {
        pointer-events: none;
        opacity: 0;
        transition: opacity 200ms ${cssVar$1.motionEaseOut};
      }

      &:hover {
        time,
        div[role='menubar'] {
          pointer-events: unset;
          opacity: 1;
        }
      }

      div[role='menubar']:has(.lobe-dropdown-menu-trigger[data-popup-open]) {
        pointer-events: unset;
        opacity: 1;
      }

      ${responsive.sm} {
        padding-block-start: 16px;
        padding-inline: 8px;
      }
    `,
		editingContainer: css$1`
      ${editingStylish}
      padding-block: 8px 12px;
      padding-inline: 12px;
      border: 1px solid ${cssVar$1.colorBorderSecondary};

      &:active,
      &:hover {
        border-color: ${cssVar$1.colorBorder};
      }
    `,
		editingContainerDocs: css$1`
      ${editingStylish}
      padding-block: 8px 12px;
      padding-inline: 12px;
      border: 1px solid ${cssVar$1.colorBorderSecondary};
      border-radius: ${cssVar$1.borderRadius};

      background: ${cssVar$1.colorFillQuaternary};

      &:active,
      &:hover {
        border-color: ${cssVar$1.colorBorder};
      }
    `,
		editingInput: css$1`
      width: 100%;
    `,
		errorContainer: css$1`
      position: relative;
      overflow: hidden;
      width: 100%;
    `,
		loadingLeft: css$1`
      position: absolute;
      inset-block-end: 0;
      inset-inline-start: -4px;

      width: 16px;
      height: 16px;
      border-radius: 50%;

      color: ${cssVar$1.colorBgLayout};

      background: ${cssVar$1.colorPrimary};
    `,
		loadingRight: css$1`
      position: absolute;
      inset-block-end: 0;
      inset-inline-end: -4px;

      width: 16px;
      height: 16px;
      border-radius: 50%;

      color: ${cssVar$1.colorBgLayout};

      background: ${cssVar$1.colorPrimary};
    `,
		messageBubble: css$1`
      ${blockStylish}
      position: relative;
      overflow: hidden;
      max-width: 100%;

      ${responsive.sm} {
        width: 100%;
      }
    `,
		messageContainer: css$1`
      position: relative;
      overflow: hidden;
      max-width: 100%;

      ${responsive.sm} {
        overflow-x: auto;
      }
    `,
		messageContainerEditing: css$1`
      ${editingStylish}
      position: relative;
      overflow: hidden;
      max-width: 100%;

      ${responsive.sm} {
        overflow-x: auto;
      }
    `,
		messageContainerEditingWithTime: css$1`
      ${editingStylish}
      position: relative;
      overflow: hidden;
      max-width: 100%;
      margin-block-start: -16px;

      ${responsive.sm} {
        overflow-x: auto;
      }
    `,
		messageContainerWithTime: css$1`
      position: relative;
      overflow: hidden;
      max-width: 100%;
      margin-block-start: -16px;

      ${responsive.sm} {
        overflow-x: auto;
      }
    `,
		messageContent: css$1`
      position: relative;
      overflow: hidden;
      max-width: 100%;

      ${responsive.sm} {
        flex-direction: column !important;
      }
    `,
		messageContentEditing: css$1`
      ${editingStylish}
      position: relative;
      overflow: hidden;
      max-width: 100%;

      ${responsive.sm} {
        flex-direction: column !important;
      }
    `,
		messageDocsWithTitle: css$1`
      ${rawStylishWithTitle}
      position: relative;
      overflow: hidden;
      max-width: 100%;

      ${responsive.sm} {
        width: 100%;
      }
    `,
		messageDocsWithoutTitle: css$1`
      ${rawStylishWithoutTitle}
      position: relative;
      overflow: hidden;
      max-width: 100%;

      ${responsive.sm} {
        width: 100%;
      }
    `,
		messageExtra: css$1`
      /* message-extra class */
    `,
		nameLeft: css$1`
      pointer-events: none;

      margin-block-end: 6px;

      font-size: 12px;
      line-height: 1;
      color: ${cssVar$1.colorTextDescription};
      text-align: start;
    `,
		nameRight: css$1`
      pointer-events: none;

      margin-block-end: 6px;

      font-size: 12px;
      line-height: 1;
      color: ${cssVar$1.colorTextDescription};
      text-align: end;
    `
	};
});

//#endregion
export { styles };
//# sourceMappingURL=style.mjs.map