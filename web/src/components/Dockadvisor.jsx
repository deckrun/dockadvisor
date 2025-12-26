"use client"

import {useEffect, useState} from "react";
import React, {useRef} from "react";
import Editor, {useMonaco} from "@monaco-editor/react";
import {
  DocumentMagnifyingGlassIcon,
  ArrowTopRightOnSquareIcon,
  DocumentCheckIcon,
  ClipboardDocumentListIcon,
  ClipboardDocumentIcon,
  ClipboardDocumentCheckIcon,
  ChevronDownIcon,
  ShareIcon
} from '@heroicons/react/24/outline'
import {LockClosedIcon, ExclamationTriangleIcon} from '@heroicons/react/16/solid'
import {ExclamationCircleIcon} from '@heroicons/react/24/solid'
import {ProgressBar} from "react-loader-spinner";
import {Menu, MenuButton, MenuItem, MenuItems} from '@headlessui/react'
import LZString from 'lz-string'

// Example Dockerfile templates
const EXAMPLES = [
  {
    id: 'bad-nodejs',
    name: '❌ Bad Node.js',
    description: 'Dockerfile with common mistakes',
    content: `FROM node
    
# Set working directory
WORKDIR app

# Copy application files
COPY . .

# Install dependencies
RUN npm install

# Expose application port
EXPOSE 3000/TCP

# Start the application
CMD npm start`
  },
  {
    id: 'good-nodejs',
    name: '✅ Optimized Node.js',
    description: 'Multi-stage build with layer caching',
    content: `# Build stage
FROM node:20-alpine AS builder

# Set working directory
WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --only=production && npm cache clean --force

# Copy source code
COPY . .

# Production stage
FROM node:20-alpine AS production

# Create app directory
WORKDIR /app

# Create non-root user
RUN addgroup -g 1001 -S nodejs && \\
    adduser -S nextjs -u 1001

# Copy built application from builder stage
COPY --from=builder --chown=nextjs:nodejs /app /app

# Switch to non-root user
USER nextjs

# Expose port
EXPOSE 8080

# Add health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \\
  CMD node -e "require('http').get('http://localhost:8080', (res) => { process.exit(res.statusCode === 200 ? 0 : 1) })"

# Start the application
CMD ["node", "app.js"]`
  },
  {
    id: 'go-multistage',
    name: '🚀 Go Multi-stage',
    description: 'Minimal Go image with multi-stage build',
    content: `# syntax=docker/dockerfile:1

# Build stage
ARG GO_VERSION=1.25

FROM golang:\${GO_VERSION}-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o app

# Production stage
FROM alpine:latest

WORKDIR /

RUN apk --no-cache add ca-certificates

COPY --from=builder /build/app /app

EXPOSE 8080

CMD [ "/app" ]`
  },
];

export function DockadvisorEditor({score, setScore, isEmpty, setIsEmpty, isCalculating, setIsCalculating, editorReady, setEditorReady}) {
  const [wasmReady, setWasmReady] = useState(false);
  const [wasmError, setWasmError] = useState(null);
  const [rules, setRules] = useState([]);
  const [copied, setCopied] = useState(false);
  const [linkCopied, setLinkCopied] = useState(false);
  const monaco = useMonaco();
  const editorRef = useRef(null);
  const decorationIds = useRef([]);

  const goToLine = (lineNumber) => {
    if (editorRef.current) {
      editorRef.current.setPosition({lineNumber, column: 1});
      editorRef.current.revealLineInCenter(lineNumber);
      editorRef.current.focus();
    }
  };

  const copyToClipboard = async () => {
    if (editorRef.current) {
      const content = editorRef.current.getValue();
      try {
        await navigator.clipboard.writeText(content);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      } catch (err) {
        console.error('Failed to copy:', err);
      }
    }
  };

  const generateShareLink = async () => {
    if (editorRef.current) {
      const content = editorRef.current.getValue();
      try {
        // Compress the Dockerfile content
        const compressed = LZString.compressToEncodedURIComponent(content);
        // Generate the share URL
        const shareUrl = `${window.location.origin}${window.location.pathname}?d=${compressed}`;
        // Copy to clipboard
        await navigator.clipboard.writeText(shareUrl);
        setLinkCopied(true);
        setTimeout(() => setLinkCopied(false), 2000);
      } catch (err) {
        console.error('Failed to generate share link:', err);
      }
    }
  };

  const loadExample = (exampleContent) => {
    if (editorRef.current) {
      editorRef.current.setValue(exampleContent);
      editorRef.current.focus();
    }
  };


  useEffect(() => {
    async function loadWasmExecScript() {
      // Load wasm_exec.js dynamically
      return new Promise((resolve, reject) => {
        // Check if script is already loaded
        if (window.Go) {
          resolve();
          return;
        }

        const script = document.createElement('script');
        script.src = '/js/wasm_exec.js';
        script.async = true;

        script.onload = () => {
          resolve();
        };

        script.onerror = () => {
          reject(new Error('Failed to load wasm_exec.js script'));
        };

        document.head.appendChild(script);
      });
    }

    async function loadWasm() {
      try {
        // Check if WASM is already loaded or loading (prevents double loading in React Strict Mode)
        if (window.parseDockerfile) {
          setWasmReady(true);
          return;
        }

        // Use a global flag to prevent concurrent loading attempts
        if (window.__dockadvisorWasmLoading) {
          // Wait for the other load to complete
          const checkInterval = setInterval(() => {
            if (window.parseDockerfile) {
              clearInterval(checkInterval);
              setWasmReady(true);
            }
          }, 100);
          return;
        }
        window.__dockadvisorWasmLoading = true;

        // Load the wasm_exec.js script first
        await loadWasmExecScript();

        // Wait a bit for window.Go to be available with timeout
        const timeout = 5000; // 5 seconds (shorter since script is already loaded)
        const startTime = Date.now();

        while (!window.Go) {
          if (Date.now() - startTime > timeout) {
            throw new Error('Failed to initialize WebAssembly runtime. Please try refreshing the page.');
          }
          await new Promise(resolve => setTimeout(resolve, 50));
        }

        const go = new window.Go();
        const resp = await fetch("/js/dockadvisor.wasm");

        if (!resp.ok) {
          throw new Error(`Failed to fetch WebAssembly module: ${resp.statusText}`);
        }

        const buf = await resp.arrayBuffer();
        const {instance} = await WebAssembly.instantiate(buf, go.importObject);

        // Start the Go WASM runtime and handle any errors
        // Note: go.run() returns a Promise that resolves when the Go program exits
        // For modules that expose JS functions, the program runs indefinitely
        go.run(instance).catch(err => {
          console.error('Go WASM runtime error:', err);
          // Convert object errors to string for better error messages
          let errorMessage = 'WebAssembly runtime error';
          if (err && typeof err === 'object') {
            if (err.message) {
              errorMessage = err.message;
            } else if (typeof err.toString === 'function') {
              const str = err.toString();
              if (str && str !== '[object Object]') {
                errorMessage = str;
              }
            }
          } else if (typeof err === 'string') {
            errorMessage = err;
          }
          setWasmError(errorMessage);
        });

        setWasmReady(true);
      } catch (err) {
        console.error('WebAssembly loading error:', err);
        // Handle different error types
        let errorMessage = 'Failed to load analyzer. Please refresh the page.';
        if (err && typeof err === 'object') {
          if (err.message) {
            errorMessage = err.message;
          } else if (typeof err.toString === 'function') {
            const str = err.toString();
            if (str && str !== '[object Object]') {
              errorMessage = str;
            }
          }
        } else if (typeof err === 'string') {
          errorMessage = err;
        }
        setWasmError(errorMessage);
      }
    }

    loadWasm();
  }, []);

  useEffect(() => {
    if (monaco) {
      monaco.editor.defineTheme("my-dark", {
        base: "vs",
        inherit: true,
        rules: [],
        colors: {
          "editor.background": "#f9fafb"
        }
      });
    }
  }, [monaco]);

  // Load Dockerfile from URL parameter on mount
  useEffect(() => {
    if (editorRef.current && editorReady) {
      const params = new URLSearchParams(window.location.search);
      const compressed = params.get('d');

      if (compressed) {
        try {
          const decompressed = LZString.decompressFromEncodedURIComponent(compressed);
          if (decompressed) {
            editorRef.current.setValue(decompressed);
            // Clear URL parameter for cleaner URL
            window.history.replaceState({}, document.title, window.location.pathname);
          }
        } catch (err) {
          console.error('Failed to load Dockerfile from URL:', err);
        }
      }
    }
  }, [editorReady]);

  function getEditorContentListener(editor, monaco) {
    return () => {
      const content = editor.getValue();
      const wasEmpty = !content || content.trim() === '';

      // Update isEmpty immediately (no debounce) for instant placeholder hide/show
      if (wasEmpty) {
        setIsEmpty(true);
        setIsCalculating(false);
      } else {
        // Only set calculating on first paste (empty -> content transition)
        setIsEmpty((prevIsEmpty) => {
          if (prevIsEmpty) {
            setIsCalculating(true);
          }
          return false;
        });
      }

      // Debounce the parsing logic
      clearTimeout(window.dockerfileParseTimeout);
      window.dockerfileParseTimeout = setTimeout(() => {
        const currentContent = editor.getValue();

        // Check if editor is empty
        if (!currentContent || currentContent.trim() === '') {
          setRules([]);
          setScore(100);
          setIsCalculating(false);
          decorationIds.current = editor.deltaDecorations(decorationIds.current, []);
          return;
        }

        const result = window.parseDockerfile(currentContent);
        if (!result.success) {
          // Show parse error as a critical rule
          const errorRule = {
            code: 'PARSE_ERROR',
            description: result.error || 'Failed to parse Dockerfile. Please check your syntax.',
            startLine: 1,
            endLine: 1,
            url: null,
          };
          setRules([errorRule]);
          setScore(0); // Parse error = 0 score
          setIsCalculating(false);
          decorationIds.current = editor.deltaDecorations(decorationIds.current, []);
          return
        }

        const currentRules = result.rules || [];
        // Sort rules by line number
        const sortedRules = currentRules.sort((a, b) => a.startLine - b.startLine);
        setRules(sortedRules);
        setScore(result.score ?? 100);
        setIsCalculating(false);

        const decorations = result.rules.map(rule => ({
          range: new monaco.Range(rule.startLine, 1, rule.endLine, 1),
          options: {
            isWholeLine: true,
            className: rule.severity === 'error' || rule.severity === 'fatal' ? 'cm-line-highlight-error' : 'cm-line-highlight-warning',
            hoverMessage: {value: rule.description},
          },
        }));
        decorationIds.current = editor.deltaDecorations(decorationIds.current, decorations);
      }, 250); // Wait 250ms after user stops typing
    };
  }

  const handleMount = (editor, monaco) => {
    editorRef.current = editor;

    const model = monaco.editor.createModel('', "dockerfile");
    editor.setModel(model);

    // Listen to content changes
    model.onDidChangeContent(getEditorContentListener(editor, monaco));

    setEditorReady(true);
  };

  return (
    <div>
        {wasmError && (
          <div className="mt-6 p-6 bg-red-50 border border-red-200 rounded-lg">
            <div className="flex gap-3">
              <div className="flex-shrink-0">
                <ExclamationCircleIcon className="size-6 text-red-500"/>
              </div>
              <div className="flex-1">
                <h3 className="text-lg font-semibold text-red-900 mb-2">
                  Failed to Load Analyzer
                </h3>
                <p className="text-base text-red-700 mb-3">
                  {wasmError}
                </p>
                <button
                  onClick={() => window.location.reload()}
                  className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700 transition-colors"
                >
                  Reload Page
                </button>
              </div>
            </div>
          </div>
        )}
        {!wasmError && !wasmReady && (
          <div className={"flex justify-center items-center mt-6"}>
            <ProgressBar
              visible={true}
              height="100"
              width="100"
              color="#9b6ffc"
              borderColor="#7515f9"
              barColor="#9b6ffc"
              ariaLabel="progress-bar-loading"
              wrapperStyle={{}}
              wrapperClass=""
            />
          </div>
        )}
        {wasmReady && !wasmError && (
          <>
            <div
              style={{display: editorReady ? 'flex' : 'none'}}
              className="flex flex-col lg:flex-row gap-4"
            >
              {/* Left column */}
              <div className="w-full lg:w-2/3 divide-y divide-gray-200 bg-gray-50 shadow-sm rounded-lg">
                {/* Header */}
                <div className="px-4 py-3 sm:px-6 flex items-center gap-1 sm:gap-2 bg-slate-800 text-white rounded-t-lg">
                  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none"
                       stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
                       className="lucide lucide-code-xml h-4 w-4 text-muted-foreground">
                    <path d="m18 16 4-4-4-4"></path>
                    <path d="m6 8-4 4 4 4"></path>
                    <path d="m14.5 4-5 16"></path>
                  </svg>
                  <span className="text-sm font-medium text-foreground">Dockerfile</span>
                  <Menu as="div" className="relative ml-auto">
                    <MenuButton
                      className="flex items-center gap-1.5 px-2 sm:px-3 py-1 text-xs font-medium text-white bg-slate-700 hover:bg-slate-600 rounded transition-colors"
                      title="Load Example">
                      <span className="hidden sm:inline">Load Example</span>
                      <span className="sm:hidden">Load</span>
                      <ChevronDownIcon className="size-4"/>
                    </MenuButton>
                    <MenuItems
                      className="absolute left-1/2 -translate-x-1/2 sm:left-0 sm:translate-x-0 mt-2 w-72 bg-white rounded-lg shadow-lg border border-gray-200 z-50 focus:outline-none">
                      {EXAMPLES.map((example) => (
                        <MenuItem key={example.id}>
                          {({focus}) => (
                            <button
                              onClick={() => loadExample(example.content)}
                              className={`${focus ? 'bg-slate-50' : ''} w-full text-left px-4 py-3 border-b border-gray-100 last:border-0 first:rounded-t-lg last:rounded-b-lg transition-colors`}
                            >
                              <div className="font-medium text-sm text-gray-900">{example.name}</div>
                              <div className="text-xs text-gray-500 mt-1">{example.description}</div>
                            </button>
                          )}
                        </MenuItem>
                      ))}
                    </MenuItems>
                  </Menu>
                  <button
                    onClick={copyToClipboard}
                    disabled={isEmpty}
                    className="flex items-center gap-1.5 px-2 sm:px-3 py-1 text-xs font-medium text-white bg-slate-700 hover:bg-slate-600 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    title={copied ? "Copied!" : "Copy to clipboard"}
                  >
                    {copied ? (
                      <>
                        <ClipboardDocumentCheckIcon className="size-4"/>
                        <span>Copied!</span>
                      </>
                    ) : (
                      <>
                        <ClipboardDocumentIcon className="size-4"/>
                        <span>Copy</span>
                      </>
                    )}
                  </button>
                  <button
                    onClick={generateShareLink}
                    disabled={isEmpty}
                    className="flex items-center gap-1.5 px-2 sm:px-3 py-1 text-xs font-medium text-white bg-slate-700 hover:bg-slate-600 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    title={linkCopied ? "Link Copied!" : "Share link"}
                  >
                    {linkCopied ? (
                      <>
                        <ShareIcon className="size-4"/>
                        <span className="hidden sm:inline">Link Copied!</span>
                        <span className="sm:hidden">Copied!</span>
                      </>
                    ) : (
                      <>
                        <ShareIcon className="size-4"/>
                        <span className="hidden sm:inline">Share Link</span>
                        <span className="sm:hidden">Share</span>
                      </>
                    )}
                  </button>
                </div>
                <div className="px-0 py-4 relative rounded-b-lg overflow-hidden">
                  <Editor
                    height="500px"
                    defaultLanguage="dockerfile"
                    theme="my-dark"
                    onMount={handleMount}
                    options={{
                      fontSize: 14,
                      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
                      lineHeight: 24,
                      automaticLayout: true,
                      minimap: {enabled: false},
                      renderLineHighlight: 'gutter',
                      renderValidationDecorations: 'on',

                      // Reduce gutter size and lateral spacing
                      glyphMargin: false,
                      folding: false,
                      lineNumbers: 'on',
                      lineNumbersMinChars: 3,
                      lineDecorationsWidth: 12,
                      inlineSuggest: {enabled: false}
                    }}
                  />
                  <div
                    className="monaco-placeholder"
                    style={{
                      display: isEmpty ? 'block' : 'none',
                      position: 'absolute',
                      top: '16px',
                      left: '40px',
                      fontSize: '14px',
                      lineHeight: '24px',
                      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
                      color: '#9CA3AF',
                      pointerEvents: 'none',
                      userSelect: 'none',
                      whiteSpace: 'pre-wrap'
                    }}
                  >
                    # Paste your Dockerfile here...
                  </div>
                  <style>{`
                .cm-line-highlight-warning { background: rgba(251, 191, 36, 0.2); }
                .cm-line-highlight-error { background: rgba(239, 68, 68, 0.2); }
                .cm-glyph-error { background: url('data:image/svg+xml;utf8,<svg .../>') center/contain no-repeat; width:16px; height:16px; }
            `}</style>
                </div>

              </div>

              {/* Right column */}
              <div
                className="w-full lg:w-1/3 flex flex-col divide-y divide-gray-200 overflow-hidden rounded-lg bg-gray-50 shadow-sm h-[400px] lg:h-[581px]">

                {/* Header */}
                <div className="px-4 py-3 sm:px-6 flex items-center gap-2 bg-slate-800 text-white">
                  <DocumentMagnifyingGlassIcon className="size-6"/>
                  <span className="text-sm font-medium text-foreground">Analysis Results</span>
                </div>

                {/* Content */}
                <div className="flex-1 flex flex-col overflow-hidden">

                  {/* Rules */}
                  {rules.length === 0 ? (
                    <div
                      className="px-4 py-4 space-y-3 flex-1 flex flex-col items-center justify-center">
                      <div className="text-gray-500 flex flex-col items-center justify-center">
                        {isEmpty ? (
                          <>
                            <div className="flex items-center justify-center size-20 rounded-full bg-blue-100 mb-4">
                              <ClipboardDocumentListIcon className="size-12 text-blue-600"/>
                            </div>
                            <p className="text-sm font-medium">Paste your Dockerfile to see issues</p>
                          </>
                        ) : (
                          <>
                            <div className="flex items-center justify-center size-20 rounded-full bg-green-100 mb-4">
                              <DocumentCheckIcon className="size-12 text-green-600"/>
                            </div>
                            <p className="text-sm font-medium text-green-700">No issues found</p>
                          </>
                        )}
                      </div>
                    </div>
                  ) : (
                    <>
                      <div
                        className="flex-1 min-h-0 px-4 py-4 space-y-3 overflow-hidden overflow-y-auto">
                        <p className="text-gray-500 text-sm font-medium">{rules.length} issues found</p>
                        {rules.map((rule, index) => (
                          <RuleWarning
                            key={index}
                            rule={rule}
                            onGoToLine={goToLine}
                          />
                        ))}
                      </div>
                    </>
                  )}

                </div>
              </div>
            </div>
            <div className="pl-1 py-2 mb-10 flex items-center gap-1">
              <LockClosedIcon className="size-4 text-gray-700"/>
              <span className="text-xs">We don’t send your Dockerfile anywhere. Everything runs client-side.</span>
            </div>
          </>
        )}
    </div>
  );
}

export function ScoreGauge({score, isEmpty, isCalculating}) {
  // Show empty state if empty or calculating
  const showEmptyState = isEmpty || isCalculating;

  // Determine color based on Lighthouse rules or gray if empty/calculating
  const getColor = (score, showEmptyState) => {
    if (showEmptyState) return {stroke: '#9ca3af', fill: '#f3f4f6'}; // gray-400, gray-100
    if (score >= 90) return {stroke: '#10b981', fill: '#d1fae5'}; // green-500, green-100
    if (score >= 50) return {stroke: '#f59e0b', fill: '#fef3c7'}; // amber-500, amber-100
    return {stroke: '#ef4444', fill: '#fee2e2'}; // red-500, red-100
  };

  const colors = getColor(score, showEmptyState);
  const radius = 45;
  const strokeWidth = 8;
  const normalizedRadius = radius - strokeWidth / 2;
  const circumference = normalizedRadius * 2 * Math.PI;
  const strokeDashoffset = showEmptyState ? circumference : circumference - (score / 100) * circumference;

  return (
    <div className="flex flex-col items-center justify-center">
      <div className="relative inline-flex items-center justify-center">
        <svg
          height={radius * 2}
          width={radius * 2}
          className="transform -rotate-90"
        >
          {/* Background circle (gray) */}
          <circle
            stroke="#e5e7eb"
            fill="transparent"
            strokeWidth={strokeWidth}
            r={normalizedRadius}
            cx={radius}
            cy={radius}
          />
          {/* Filled background circle */}
          <circle
            fill={colors.fill}
            r={normalizedRadius - strokeWidth / 2}
            cx={radius}
            cy={radius}
            style={{
              transition: 'fill 0.3s ease-in-out',
            }}
          />
          {/* Progress circle */}
          <circle
            stroke={colors.stroke}
            fill="transparent"
            strokeWidth={strokeWidth}
            strokeDasharray={circumference + ' ' + circumference}
            style={{
              strokeDashoffset,
              transition: 'stroke-dashoffset 0.5s ease-in-out, stroke 0.3s ease-in-out',
            }}
            strokeLinecap="round"
            r={normalizedRadius}
            cx={radius}
            cy={radius}
          />
        </svg>
        {/* Score text */}
        <div className="absolute inset-0 flex items-center justify-center">
          <span className="text-2xl font-bold" style={{
            color: colors.stroke,
            fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace'
          }}>
            {showEmptyState ? '—' : score}
          </span>
        </div>
      </div>
      <div className="mt-1.5 text-base font-semibold text-gray-700">
        Overall Score
      </div>
    </div>
  );
}

function RuleWarning({rule, onGoToLine}) {
  const handleClick = (e) => {
    // Don't trigger if clicking on the learn more link
    if (e.target.closest('a')) {
      return;
    }
    onGoToLine(rule.startLine);
  };

  const isError = rule.severity === 'error' || rule.severity === 'fatal';
  const Icon = isError ? ExclamationCircleIcon : ExclamationTriangleIcon;
  const iconColor = isError ? 'text-red-500' : 'text-yellow-500';
  const hoverBorderColor = isError ? 'hover:border-red-400' : 'hover:border-yellow-400';

  return (
    <div
      onClick={handleClick}
      className={`flex gap-3 p-3 bg-white rounded-lg border border-gray-200 ${hoverBorderColor} hover:shadow-md transition-all cursor-pointer`}
    >
      <div className="flex-shrink-0">
        <Icon className={`size-6 ${iconColor}`}/>
      </div>
      <div className="flex-1 min-w-0">
        <h4 className="text-sm font-semibold text-gray-900 mb-1">
          {rule.code}
        </h4>
        <p className="text-sm text-gray-600 mb-2">
          {rule.description}
        </p>
        <div className="flex items-center justify-between gap-2">
          <span className="text-xs text-gray-400 font-medium">
            Line {rule.startLine}
          </span>
          {rule.url && (
            <a
              href={rule.url}
              target="_blank"
              rel="noopener noreferrer"
              onClick={(e) => e.stopPropagation()}
              className="text-xs text-blue-600 hover:text-blue-800 hover:underline font-medium flex items-center gap-1"
            >
              Learn more
              <ArrowTopRightOnSquareIcon className="size-3"/>
            </a>
          )}
        </div>
      </div>
    </div>
  );
}

