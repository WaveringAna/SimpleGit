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
