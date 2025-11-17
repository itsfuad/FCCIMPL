import type { APIRoute } from 'astro';
import { execFile } from 'child_process';
import { promisify } from 'util';
import { writeFile, unlink } from 'fs/promises';
import { join } from 'path';
import { tmpdir } from 'os';

export const prerender = false; // This is a server-side route

const execFileAsync = promisify(execFile);

// ANSI color code to HTML conversion
function ansiToHtml(text: string): string {
  // ANSI color codes mapping
  const ansiColors: Record<string, string> = {
    '0': 'reset',
    '1': 'bold',
    '30': '#000000', '90': '#808080',  // Black, Bright Black
    '31': '#ef4444', '91': '#f87171',  // Red, Bright Red
    '32': '#10b981', '92': '#34d399',  // Green, Bright Green
    '33': '#f59e0b', '93': '#fbbf24',  // Yellow, Bright Yellow
    '34': '#3b82f6', '94': '#60a5fa',  // Blue, Bright Blue
    '35': '#a855f7', '95': '#c084fc',  // Magenta, Bright Magenta
    '36': '#06b6d4', '96': '#22d3ee',  // Cyan, Bright Cyan
    '37': '#d1d5db', '97': '#f3f4f6',  // White, Bright White
  };

  let html = '';
  let currentColor = '';
  let isBold = false;
  
  // Escape HTML
  text = text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
  
  // Parse ANSI codes
  const parts = text.split(/\x1b\[([0-9;]+)m/);
  
  for (let i = 0; i < parts.length; i++) {
    if (i % 2 === 0) {
      // Text content
      if (parts[i]) {
        if (currentColor || isBold) {
          const style = [];
          if (currentColor) style.push(`color: ${currentColor}`);
          if (isBold) style.push('font-weight: bold');
          html += `<span style="${style.join('; ')}">${parts[i]}</span>`;
        } else {
          html += parts[i];
        }
      }
    } else {
      // ANSI code
      const codes = parts[i].split(';');
      for (const code of codes) {
        if (code === '0') {
          currentColor = '';
          isBold = false;
        } else if (code === '1') {
          isBold = true;
        } else if (ansiColors[code]) {
          currentColor = ansiColors[code];
        }
      }
    }
  }
  
  return html;
}

export const POST: APIRoute = async (context) => {
  let tempFile: string | null = null;
  
  try {
    // Try to get the request body
    const contentType = context.request.headers.get('content-type');
    console.log('Content-Type:', contentType);
    
    let code: string;
    
    if (contentType?.includes('application/json')) {
      try {
        const body = await context.request.json();
        code = body.code;
      } catch (e) {
        console.error('JSON parse error:', e);
        return new Response(JSON.stringify({ 
          success: false,
          error: `Failed to parse JSON: ${e instanceof Error ? e.message : 'Unknown'}`
        }), {
          status: 400,
          headers: { 'Content-Type': 'application/json' }
        });
      }
    } else {
      return new Response(JSON.stringify({ 
        success: false,
        error: 'Content-Type must be application/json'
      }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' }
      });
    }

    if (!code || typeof code !== 'string') {
      return new Response(JSON.stringify({ 
        success: false,
        error: 'Invalid code provided - must be a non-empty string' 
      }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' }
      });
    }

    if (code.trim().length == 0) {
      return new Response(JSON.stringify({
        success: false,
        error: 'Code is empty'
      }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' }
      });
    }

    // Create temporary file
    tempFile = join(tmpdir(), `ferret-${Date.now()}.fer`);
    await writeFile(tempFile, code, 'utf-8');
    console.log('Temp file created:', tempFile);

    // Path to compiler in public folder
    const compilerPath = join(process.cwd(), 'public', 'ferret.exe');
    console.log('Compiler path:', compilerPath);

    try {
      const { stdout, stderr } = await execFileAsync(compilerPath, [tempFile], {
        timeout: 10000,
        maxBuffer: 1024 * 1024
      });

      console.log('Execution successful');
      return new Response(JSON.stringify({
        success: true,
        output: stdout || 'Program executed successfully (no output)',
        html: ansiToHtml(stdout || 'Program executed successfully (no output)')
      }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' }
      });

    } catch (execError: any) {

      const errorMessage = execError.stderr || execError.message || 'Compilation failed';
      const outputMessage = execError.stdout || '';
      
      return new Response(JSON.stringify({
        success: false,
        error: errorMessage,
        output: outputMessage,
        errorHtml: ansiToHtml(errorMessage),
        outputHtml: outputMessage ? ansiToHtml(outputMessage) : ''
      }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' }
      });
    }

  } catch (error: any) {
    console.error('Server error:', error);
    return new Response(JSON.stringify({ 
      success: false,
      error: `Server error: ${error.message || 'Unknown error'}`,
      stack: error.stack
    }), {
      status: 500,
      headers: { 'Content-Type': 'application/json' }
    });
  } finally {
    // Always clean up temp file
    if (tempFile) {
      try {
        await unlink(tempFile);
      } catch (cleanupError) {
        console.error('Failed to delete temp file:', cleanupError);
      }
    }
  }
};
