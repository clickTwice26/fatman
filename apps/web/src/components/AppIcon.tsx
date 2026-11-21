"use client";

import { HugeiconsIcon, type IconSvgElement } from "@hugeicons/react";

export interface AppIconProps {
  icon: IconSvgElement;
  size?: number;
  strokeWidth?: number;
  color?: string;
  label?: string;
}

export function AppIcon({
  icon,
  size = 20,
  strokeWidth = 1.8,
  color = "currentColor",
  label,
}: AppIconProps) {
  return (
    <HugeiconsIcon
      icon={icon}
      size={size}
      strokeWidth={strokeWidth}
      color={color}
      aria-hidden={label ? undefined : true}
      aria-label={label}
      role={label ? "img" : undefined}
    />
  );
}
