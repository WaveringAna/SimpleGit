import express from 'express';
import cors from 'cors';
import { Highlighter } from './highlighter';
import { HighlightRequest } from './types';

const app = express();
const port = process.env.TS_SERVICE_PORT || 3001;
const highlighter = new Highlighter();

app.use(cors());
app.use(express.json());

app.post('/highlight', (req, res) => {
  try {
    const request = req.body as HighlightRequest;
    const result = highlighter.highlight(request);
    res.json(result);
  } catch (error) {
    console.error('Highlight error:', error);
    res.status(500).json({ error: 'Highlighting failed' });
  }
});

app.get('/health', (req, res) => {
  res.json({ status: 'healthy' });
});

app.listen(port, () => {
  console.log(`TypeScript service listening on port ${port}`);
});

