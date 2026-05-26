import ReactMarkdown from "react-markdown";
import rehypeSanitize from "rehype-sanitize";
import rehypeSlug from "rehype-slug";
import remarkGfm from "remark-gfm";

type MarkdownContentProps = {
  markdown: string;
  compact?: boolean;
  className?: string;
};

export default function MarkdownContent({
  markdown,
  compact = false,
  className = "",
}: MarkdownContentProps) {
  return (
    <div
      className={`markdown-content ${compact ? "markdown-content-compact" : ""} ${className}`}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeSanitize, rehypeSlug]}
        components={{
          a: ({ children, ...props }) => (
            <a {...props} target="_blank" rel="noreferrer">
              {children}
            </a>
          ),
        }}
      >
        {markdown}
      </ReactMarkdown>
    </div>
  );
}
