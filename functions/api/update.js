/**
 * Cloudflare Worker to receive status updates from backend
 * This endpoint is called by the backend to push status updates
 */

export async function onRequestPost(context) {
  const { request, env } = context;

  try {
    // Validate API key
    const authHeader = request.headers.get('Authorization');
    const expectedAuth = `Bearer ${env.API_KEY}`;

    if (!authHeader || authHeader !== expectedAuth) {
      return new Response(JSON.stringify({ error: 'Unauthorized' }), {
        status: 401,
        headers: { 'Content-Type': 'application/json' }
      });
    }

    // Parse request body
    const data = await request.json();

    // Validate timestamp (reject if older than 10 minutes)
    if (data.last_update_time) {
      const updateTime = new Date(data.last_update_time);
      const now = new Date();
      const ageMinutes = (now - updateTime) / 1000 / 60;

      if (ageMinutes > 10) {
        return new Response(JSON.stringify({ error: 'Timestamp too old' }), {
          status: 400,
          headers: { 'Content-Type': 'application/json' }
        });
      }
    }

    // Store in KV
    await env.STATUS_STORE.put('current_status', JSON.stringify(data));

    return new Response(JSON.stringify({ status: 'ok', message: 'Status updated successfully' }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' }
    });
  } catch (error) {
    console.error('Error processing update:', error);
    return new Response(JSON.stringify({ error: 'Internal server error' }), {
      status: 500,
      headers: { 'Content-Type': 'application/json' }
    });
  }
}
