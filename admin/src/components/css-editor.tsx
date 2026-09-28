"use client";

import dynamic from "next/dynamic";
import { useRef } from "react";
import type { OnMount } from "@monaco-editor/react";
import type { editor, languages, Position } from "monaco-editor";
import { Button } from "@/components/ui/button";

const MonacoEditor = dynamic(() => import("@monaco-editor/react"), {
  ssr: false,
  loading: () => (
    <div className="flex h-[560px] items-center justify-center bg-[#1e1e1e] text-sm text-white/60">
      编辑器加载中...
    </div>
  ),
});

const CSS_VARIABLES = [
  ["--page-background-color", "页面背景色"],
  ["--page-background-image", "页面背景图层、渐变或 none"],
  ["--page-background-size", "页面背景图层尺寸"],
  ["--page-background-attachment", "页面背景滚动行为"],
  ["--color-page-foreground", "页面前景文字颜色"],
  ["--color-brand", "品牌主色"],
  ["--color-brand-foreground", "品牌色上的文字颜色"],
  ["--color-nav-foreground", "导航栏文字和图标颜色"],
  ["--color-surface", "卡片表面颜色"],
  ["--color-overlay", "浮层表面颜色"],
  ["--color-control-secondary", "控件次要颜色"],
  ["--color-control-hover", "控件悬浮颜色"],
  ["--color-control-border", "控件边框颜色"],
  ["--color-input-border", "输入框边框颜色"],
  ["--color-focus-ring", "焦点环颜色"],
  ["--radius-base", "默认圆角"],
  ["--color-fabric-stitch", "Fabric 缝线颜色"],
  ["--color-fabric-shadow", "Fabric 阴影颜色"],
  ["--color-fabric-grain", "Fabric 纹理颜色"],
] as const;

interface CSSEditorProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}

export function CSSEditor({ value, onChange, disabled = false }: CSSEditorProps) {
  const editorRef = useRef<Parameters<OnMount>[0] | null>(null);

  const handleMount: OnMount = (editor, monaco) => {
    editorRef.current = editor;
    const provider: languages.CompletionItemProvider = {
      triggerCharacters: ["-"],
      provideCompletionItems(model: editor.ITextModel, position: Position) {
        const word = model.getWordUntilPosition(position);
        const range = {
          startLineNumber: position.lineNumber,
          endLineNumber: position.lineNumber,
          startColumn: word.startColumn,
          endColumn: word.endColumn,
        };
        return {
          suggestions: CSS_VARIABLES.map(([label, documentation]) => ({
            label,
            kind: monaco.languages.CompletionItemKind.Variable,
            insertText: label,
            documentation,
            range,
          })),
        };
      },
    };
    const disposable = monaco.languages.registerCompletionItemProvider("css", provider);
    editor.onDidDispose(() => disposable.dispose());
  };

  const format = () => {
    void editorRef.current?.getAction("editor.action.formatDocument")?.run();
  };

  return (
    <div className="overflow-hidden rounded-lg border border-[#3c3c3c]">
      <div className="flex items-center justify-between bg-[#252526] px-3 py-2">
        <span className="font-mono text-xs text-white/60">custom.css</span>
        <Button
          type="button"
          size="sm"
          variant="secondary"
          onClick={format}
          disabled={disabled}
        >
          格式化 CSS
        </Button>
      </div>
      <MonacoEditor
        height="560px"
        language="css"
        theme="vs-dark"
        value={value}
        onChange={(nextValue) => onChange(nextValue ?? "")}
        onMount={handleMount}
        options={{
          readOnly: disabled,
          automaticLayout: true,
          colorDecorators: true,
          formatOnPaste: true,
          formatOnType: true,
          bracketPairColorization: { enabled: true },
          minimap: { enabled: false },
          wordWrap: "on",
          tabSize: 2,
          padding: { top: 12, bottom: 12 },
          scrollBeyondLastLine: false,
          suggestOnTriggerCharacters: true,
        }}
      />
    </div>
  );
}
