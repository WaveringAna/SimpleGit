export interface HighlightRequest {
  code: string;
  language?: string;
  filename?: string;
}

export interface HighlightResponse {
  highlighted: string;
  detectedLanguage: string;
  /*symbols: Array<{
    name: string;
    type: string;
    line: number;
  }>;*/
}

export type LanguageMap = Record<string, string>;

export const DEFAULT_LANGUAGE_MAP: LanguageMap = {
  // Programming Languages
  'ts': 'typescript',
  'js': 'javascript',
  'jsx': 'javascript',
  'tsx': 'typescript',
  'py': 'python',
  'rb': 'ruby',
  'php': 'php',
  'java': 'java',
  'kt': 'kotlin',
  'go': 'go',
  'rs': 'rust',
  'swift': 'swift',
  'scala': 'scala',
  'dart': 'dart',
  'cs': 'csharp',
  'c': 'c',
  'cpp': 'cpp',
  'h': 'cpp',
  'hpp': 'cpp',
  'm': 'objectivec',
  
  // Scripting & Web
  'sh': 'bash',
  'bash': 'bash',
  'zsh': 'bash',
  'sql': 'sql',
  'html': 'html',
  'xml': 'xml',
  'css': 'css',
  'scss': 'scss',
  'less': 'less',
  
  // Configuration & Markup
  'json': 'json',
  'yaml': 'yaml',
  'yml': 'yaml',
  'toml': 'ini',
  'ini': 'ini',
  'md': 'markdown',
  'dockerfile': 'dockerfile',
  
  // Framework & Other
  'vue': 'vue',
  'lua': 'lua',
  'r': 'r',
  'pl': 'perl',
  'erl': 'erlang',
  'hrl': 'erlang'
};