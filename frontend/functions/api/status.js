/**
 * Cloudflare Worker to serve current status to frontend
 * This endpoint is polled by the frontend to get current service status
 */

export async function onRequestGet(context) {
  const { env } = context;

  try {
    // Get current status from KV
    const statusData = await env.STATUS_STORE.get('current_status');

    if (!statusData) {
      // No data yet
      return new Response(JSON.stringify({
        services: [],
        last_update_time: null
      }), {
        status: 200,
        headers: {
          'Content-Type': 'application/json',
          'Cache-Control': 'no-cache, no-store, must-revalidate',
          'Access-Control-Allow-Origin': '*'
        }
      });
    }

    // Return stored status
    return new Response(statusData, {
      status: 200,
      headers: {
        'Content-Type': 'application/json',
        'Cache-Control': 'no-cache, no-store, must-revalidate',
        'Access-Control-Allow-Origin': '*'
      }
    });
  } catch (error) {
    console.error('Error fetching status:', error);
    return new Response(JSON.stringify({ error: 'Internal server error' }), {
      status: 500,
      headers: { 'Content-Type': 'application/json' }
    });
  }
}
