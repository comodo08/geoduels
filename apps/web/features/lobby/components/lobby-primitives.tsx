import type React from "react";
import { Button, type ButtonProps } from "../../../components/ui/button";
import { Input, type InputProps } from "../../../components/ui/input";
import { Select, type SelectProps } from "../../../components/ui/select";
import { Surface, type SurfaceVariant } from "../../../components/ui/Surface";
import { Textarea, type TextareaProps } from "../../../components/ui/textarea";
import { cn } from "../../../lib/cn";

type LobbyPanelProps = {
  children: React.ReactNode;
  className?: string;
  interactive?: boolean;
  variant?: SurfaceVariant;
  style?: React.CSSProperties;
};

export function LobbyPanel({
  children,
  className,
  interactive = false,
  variant = "gameGlass",
  style,
}: LobbyPanelProps) {
  return (
    <Surface
      variant={variant}
      interactive={interactive}
      className={cn("rounded-2xl", className)}
      style={style}
    >
      {children}
    </Surface>
  );
}

export function LobbyActionButton({
  className,
  variant = "primary",
  size = "md",
  ...props
}: ButtonProps) {
  return (
    <Button
      variant={variant}
      size={size}
      className={cn("font-extrabold uppercase tracking-[0.08em]", className)}
      {...props}
    />
  );
}

export function LobbyCardButton({
  children,
  className,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { children: React.ReactNode }) {
  return (
    <button
      type="button"
      className={cn(
        "glass-panel glass-panel-interactive lobby-feature-card rounded-2xl text-left",
        className,
      )}
      {...props}
    >
      {children}
    </button>
  );
}

export function LobbyInput(props: InputProps) {
  return <Input variant="game" {...props} />;
}

export function LobbyTextarea(props: TextareaProps) {
  return <Textarea variant="game" {...props} />;
}

export function LobbySelect(props: SelectProps) {
  return <Select variant="game" {...props} />;
}

export function LobbyMutedBox(props: { children: React.ReactNode; className?: string }) {
  return (
    <Surface variant="subtle" className={cn("rounded-xl p-4 text-sm font-semibold text-inkMuted", props.className)}>
      {props.children}
    </Surface>
  );
}
