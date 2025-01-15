import { HighlightRequest, HighlightResponse } from './types';
import hljs from 'highlight.js';

export class Highlighter {
  highlight(request: HighlightRequest): HighlightResponse {
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

    return {
      highlighted: highlightedLines.join('\n'),
      detectedLanguage: language || 'plaintext'
    };
  }

  private detectLanguageFromFilename(filename: string): string {
    const ext = filename.split('.').pop()?.toLowerCase();
    const languageMap: Record<string, string> = {
      'ts': 'typescript',
      'js': 'javascript',
      'py': 'python',
      'go': 'go',
      // Add more mappings as needed
    };
    return languageMap[ext || ''] || '';
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
