"use client";

import Link from "next/link";
import Button, { type ButtonProps } from "@mui/material/Button";

export interface LinkButtonProps extends Omit<ButtonProps<typeof Link>, "component"> {
  href: string;
}

export function LinkButton({ href, ...props }: LinkButtonProps) {
  return <Button component={Link} href={href} {...props} />;
}
