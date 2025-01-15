import { HighlightRequest, HighlightResponse, LanguageMap, DEFAULT_LANGUAGE_MAP } from './types';
import hljs from 'highlight.js';

export class Highlighter {
  private languageMap: LanguageMap;

  constructor(customLanguageMap?: LanguageMap) {
    this.languageMap = customLanguageMap || DEFAULT_LANGUAGE_MAP;
  }

  private cache = new Map<string, HighlightResponse>()
  private MAX_CACHE_SIZE = 500 // Can be adjustable via config maybe

  highlight(request: HighlightRequest): HighlightResponse {
    const cacheKey = this.generateCacheKey(request);

    const cachedResult = this.cache.get(cacheKey);
    if (cachedResult)
      return cachedResult;

    let language = request.language;
    
    if (!language && request.filename) {
      language = this.detectLanguageFromFilename(request.filename);
    }

    // Split the code into lines to highlight each line separately
    const lines = request.code.split('\n');
    const highlightedLines = lines.map(line => {
      // Preserve empty lines
      if (line.trim() === '') {
        return '';
      }

      const result = language 
        ? hljs.highlight(line, { language })
        : hljs.highlightAuto(line);

      return result.value;
    });

     const result = {
      highlighted: highlightedLines.join('\n'),
      detectedLanguage: language || 'plaintext'
    };

    this.cacheResult(cacheKey, result)

    //console.log("highlighted code: ", result.highlighted)
    return result;
  }

  private generateCacheKey(request: HighlightRequest): string {
    // Create a hash-like key from the request propertites
    return `${request.filename || 'unknown'}-${request.language || 'auto'}-${this.hashCode(request.code)}`;
  }

  private hashCode(str: string): number {
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      const char = str.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash; // Convert to 32-bit integer
    }
    return Math.abs(hash);
  }

  private cacheResult(key: string, result: HighlightResponse) {
    // Implement LRU cache eviction
    if (this.cache.size >= this.MAX_CACHE_SIZE) {
      const oldestKey = this.cache.keys().next().value;
      if (oldestKey !== undefined) {
        this.cache.delete(oldestKey);
      }
    }
    this.cache.set(key, result);
  }

  private detectLanguageFromFilename(filename: string): string {
    const ext = filename.split('.').pop()?.toLowerCase();
    return this.languageMap[ext || ''] || '';
  }

  /*
  private extractSymbols(code: string, language?: string): Array<{ name: string; type: string; line: number }> {
    const symbols: Array<{ name: string; type: string; line: number }> = [];

    if (language === 'typescript' || language === 'javascript') {
      const sourceFile = ts.createSourceFile(
        'temp.ts',
        code,
        ts.ScriptTarget.Latest,
        true
      );

      const visit = (node: ts.Node) => {
        let symbol: { name: string; type: string; line: number } | undefined;

        if (ts.isFunctionDeclaration(node) && node.name) {
          symbol = {
            name: node.name.text,
            type: 'function',
            line: sourceFile.getLineAndCharacterOfPosition(node.getStart()).line + 1
          };
        } else if (ts.isClassDeclaration(node) && node.name) {
          symbol = {
            name: node.name.text,
            type: 'class',
            line: sourceFile.getLineAndCharacterOfPosition(node.getStart()).line + 1
          };
        } else if (ts.isInterfaceDeclaration(node)) {
          symbol = {
            name: node.name.text,
            type: 'interface',
            line: sourceFile.getLineAndCharacterOfPosition(node.getStart()).line + 1
          };
        }

        if (symbol) {
          symbols.push(symbol);
        }

        ts.forEachChild(node, visit);
      };

      visit(sourceFile);
    }

    return symbols;
  }*/
}
