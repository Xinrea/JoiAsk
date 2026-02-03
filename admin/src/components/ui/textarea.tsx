import * as React from "react";

import { cn } from "@/lib/utils";

function Textarea({ className, ...props }: React.ComponentProps<"textarea">) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        "border-input placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive flex field-sizing-content min-h-16 w-full rounded-lg border-[1.5px] bg-card/80 px-3 py-2 text-base shadow-[0_1px_0_rgba(255,255,255,0.4)_inset,inset_0_-1px_0_rgba(61,57,52,0.02),0_1px_2px_rgba(61,57,52,0.04)] transition-[color,box-shadow,border-color] duration-[200ms] outline-none focus-visible:ring-[3px] focus-visible:shadow-[0_1px_0_rgba(255,255,255,0.5)_inset,inset_0_-1px_0_rgba(61,57,52,0.03),0_2px_4px_rgba(61,57,52,0.06)] disabled:cursor-not-allowed disabled:opacity-50 md:text-sm",
        "hover:shadow-[0_1px_0_rgba(255,255,255,0.5)_inset,inset_0_-1px_0_rgba(61,57,52,0.025),0_1px_3px_rgba(61,57,52,0.05)] hover:border-input/80",
        className
      )}
      {...props}
    />
  );
}

export { Textarea };
