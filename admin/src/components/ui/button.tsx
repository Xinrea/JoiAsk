import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { Slot } from "radix-ui";

import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-lg text-sm font-medium transition-all duration-[200ms] ease-out disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 shrink-0 [&_svg]:shrink-0 outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive",
  {
    variants: {
      variant: {
        default:
          "bg-primary text-primary-foreground border-[1.5px] border-primary/30 shadow-[0_2px_8px_rgba(61,57,52,0.1),0_1px_0_rgba(255,255,255,0.2)_inset,inset_0_-1px_0_rgba(61,57,52,0.08)] hover:shadow-[0_4px_16px_rgba(61,57,52,0.14),0_1px_0_rgba(255,255,255,0.25)_inset,inset_0_-1px_0_rgba(61,57,52,0.1)] hover:-translate-y-1 hover:border-primary/40 active:translate-y-0 active:shadow-[inset_0_1px_3px_rgba(61,57,52,0.15),0_1px_2px_rgba(61,57,52,0.08)]",
        destructive:
          "bg-destructive text-white border-[1.5px] border-destructive/40 shadow-[0_2px_8px_rgba(184,84,80,0.25),0_1px_0_rgba(255,255,255,0.15)_inset] hover:bg-destructive/90 hover:shadow-[0_4px_12px_rgba(184,84,80,0.35),0_1px_0_rgba(255,255,255,0.2)_inset] hover:-translate-y-1 hover:border-destructive/50 active:translate-y-0 active:shadow-[inset_0_1px_3px_rgba(184,84,80,0.3)] focus-visible:ring-destructive/20 dark:focus-visible:ring-destructive/40 dark:bg-destructive/60",
        outline:
          "border-[1.5px] border-border bg-card text-foreground shadow-[0_1px_0_rgba(255,255,255,0.5)_inset,0_2px_4px_rgba(61,57,52,0.04)] hover:bg-accent hover:text-accent-foreground hover:shadow-[0_2px_8px_rgba(61,57,52,0.08),0_1px_0_rgba(255,255,255,0.6)_inset] hover:-translate-y-1 hover:border-border/80 active:translate-y-0 active:shadow-[inset_0_1px_2px_rgba(61,57,52,0.06)] dark:bg-input/30 dark:border-input dark:hover:bg-input/50",
        secondary:
          "bg-secondary text-secondary-foreground border-[1.5px] border-secondary/40 shadow-[0_1px_0_rgba(255,255,255,0.4)_inset,0_2px_4px_rgba(61,57,52,0.04)] hover:bg-secondary/90 hover:-translate-y-1 hover:shadow-[0_2px_8px_rgba(61,57,52,0.06),0_1px_0_rgba(255,255,255,0.5)_inset] active:translate-y-0",
        ghost:
          "hover:bg-accent hover:text-accent-foreground hover:-translate-y-0.5 active:translate-y-0 dark:hover:bg-accent/50",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: {
        default: "h-9 px-4 py-2 has-[>svg]:px-3",
        xs: "h-6 gap-1 rounded-md px-2 text-xs has-[>svg]:px-1.5 [&_svg:not([class*='size-'])]:size-3",
        sm: "h-8 rounded-lg gap-1.5 px-3 has-[>svg]:px-2.5",
        lg: "h-10 rounded-lg px-6 has-[>svg]:px-4",
        icon: "size-9",
        "icon-xs": "size-6 rounded-md [&_svg:not([class*='size-'])]:size-3",
        "icon-sm": "size-8",
        "icon-lg": "size-10",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);

function Button({
  className,
  variant = "default",
  size = "default",
  asChild = false,
  ...props
}: React.ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean;
  }) {
  const Comp = asChild ? Slot.Root : "button";

  return (
    <Comp
      data-slot="button"
      data-variant={variant}
      data-size={size}
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  );
}

export { Button, buttonVariants };
