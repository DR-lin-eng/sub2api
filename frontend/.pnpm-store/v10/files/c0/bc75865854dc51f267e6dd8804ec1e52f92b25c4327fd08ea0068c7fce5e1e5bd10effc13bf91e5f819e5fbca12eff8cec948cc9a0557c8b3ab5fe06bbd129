import { AProps, DivProps } from "../types/index.mjs";
import { HighlighterProps } from "../Highlighter/type.mjs";
import "../Highlighter/index.mjs";
import { MermaidProps } from "../Mermaid/type.mjs";
import "../Mermaid/index.mjs";
import { ImageProps as ImageProps$1 } from "../mdx/mdxComponents/Image.mjs";
import { CitationItem } from "../types/citation.mjs";
import { PreProps } from "../mdx/mdxComponents/Pre.mjs";
import { VideoProps } from "../mdx/mdxComponents/Video.mjs";
import "../mdx/index.mjs";
import { CSSProperties, ElementType, FC, ReactNode, Ref } from "react";
import { AnchorProps } from "antd";
import { Options } from "react-markdown";
import { Components } from "react-markdown/lib";
import { Pluggable } from "unified";

//#region src/Markdown/type.d.ts
interface TypographyProps extends DivProps {
  borderRadius?: number;
  fontSize?: number;
  headerMultiple?: number;
  lineHeight?: number;
  marginMultiple?: number;
  ref?: Ref<HTMLDivElement>;
}
interface SyntaxMarkdownProps {
  allowHtml?: boolean;
  allowHtmlList?: ElementType[];
  animated?: boolean;
  children: string;
  citations?: CitationItem[];
  componentProps?: {
    a?: Partial<AProps & AnchorProps>;
    highlight?: Partial<HighlighterProps>;
    img?: Partial<ImageProps$1>;
    mermaid?: Partial<MermaidProps>;
    pre?: Partial<PreProps>;
    video?: Partial<VideoProps>;
  };
  components?: Components & Record<string, FC>;
  enableCustomFootnotes?: boolean;
  enableGithubAlert?: boolean;
  enableLatex?: boolean;
  enableMermaid?: boolean;
  enableStream?: boolean;
  fullFeaturedCodeBlock?: boolean;
  reactMarkdownProps?: Omit<Readonly<Options>, 'components' | 'rehypePlugins' | 'remarkPlugins'>;
  rehypePlugins?: Pluggable[];
  rehypePluginsAhead?: Pluggable[];
  remarkPlugins?: Pluggable[];
  remarkPluginsAhead?: Pluggable[];
  showFootnotes?: boolean;
  variant?: 'default' | 'chat';
}
interface MarkdownProps extends SyntaxMarkdownProps, Omit<TypographyProps, 'children'> {
  className?: string;
  customRender?: (dom: ReactNode, context: {
    text: string;
  }) => ReactNode;
  enableImageGallery?: boolean;
  onDoubleClick?: () => void;
  ref?: Ref<HTMLDivElement>;
  style?: CSSProperties;
}
//#endregion
export { MarkdownProps, SyntaxMarkdownProps, TypographyProps };
//# sourceMappingURL=type.d.mts.map