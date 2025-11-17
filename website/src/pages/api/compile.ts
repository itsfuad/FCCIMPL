import type { APIRoute } from 'astro';
import { execFile } from 'child_process';
import { promisify } from 'util';
import { writeFile, unlink } from 'fs/promises';
import { join } from 'path';
import { tmpdir } from 'os';

export const prerender = false; // This is a server-side route

const execFileAsync = promisify(execFile);

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
        console.log('Parsed body:', body);
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

    console.log('Code length:', code.length);

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
        output: stdout || 'Program executed successfully (no output)'
      }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' }
      });

    } catch (execError: any) {
      console.log('Execution error:', execError);
      const errorMessage = execError.stderr || execError.message || 'Compilation failed';
      const outputMessage = execError.stdout || '';
      
      return new Response(JSON.stringify({
        success: false,
        error: errorMessage,
        output: outputMessage
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
        console.log('Temp file cleaned up');
      } catch (cleanupError) {
        console.error('Failed to delete temp file:', cleanupError);
      }
    }
  }
};
