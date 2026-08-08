// Google Maps interop for the block-location screen.
//
// The map is optional: it appears only when a key is configured in wwwroot/appsettings.json
// under "GoogleMaps:ApiKey". Without one — or when Google rejects the key, or the browser has
// no route to maps.googleapis.com — every entry point below reports failure instead of throwing,
// and the Razor page falls back to the coordinate plot it draws itself. A planner on a farm with
// no internet still gets to see where the blocks are.
(() => {
    const state = {
        loader: null,        // in-flight or settled load of the Maps API
        map: null,
        markers: new Map(),  // block id -> google.maps.Marker
        info: null,
        dotNet: null,
        authFailed: false
    };

    // Google calls this by name when the key is rejected. It fires after the script has already
    // loaded, so the promise below has resolved successfully by then and only the next render
    // can report the problem — which is why the flag is sticky.
    window.gm_authFailure = () => { state.authFailed = true; };

    function loadApi(apiKey) {
        if (state.loader) { return state.loader; }

        state.loader = new Promise((resolve, reject) => {
            if (window.google?.maps) { resolve(); return; }

            const callback = '__sgcMapsReady';
            window[callback] = () => { delete window[callback]; resolve(); };

            const script = document.createElement('script');
            script.src = 'https://maps.googleapis.com/maps/api/js'
                + `?key=${encodeURIComponent(apiKey)}&loading=async&callback=${callback}`;
            script.async = true;
            script.onerror = () => reject(new Error('The Google Maps script could not be loaded.'));
            document.head.appendChild(script);
        });

        // A failed load must not be cached, or a transient network error would disable the map
        // for the rest of the session.
        state.loader.catch(() => { state.loader = null; });
        return state.loader;
    }

    /**
     * Draws the blocks on a Google map.
     * Returns { ok: true } on success, or { ok: false, reason } when the caller should fall back.
     */
    async function render(elementId, dotNetRef, options) {
        const host = document.getElementById(elementId);
        if (!host) { return { ok: false, reason: 'The map container is not in the document.' }; }
        if (!options?.apiKey) { return { ok: false, reason: 'No Google Maps API key is configured.' }; }

        try {
            await loadApi(options.apiKey);
        } catch (e) {
            return { ok: false, reason: e.message };
        }
        if (state.authFailed) {
            return { ok: false, reason: 'Google rejected the configured Maps API key.' };
        }

        state.dotNet = dotNetRef;

        const blocks = options.blocks ?? [];
        if (blocks.length === 0) { return { ok: false, reason: 'No block has coordinates.' }; }

        if (!state.map || state.map.getDiv() !== host) {
            state.map = new google.maps.Map(host, {
                mapTypeId: options.mapTypeId || 'hybrid',   // satellite plus labels suits fields
                mapTypeControl: true,
                streetViewControl: false,
                fullscreenControl: true,
                zoom: 12,
                center: { lat: blocks[0].lat, lng: blocks[0].lng }
            });
            state.info = new google.maps.InfoWindow();
            state.markers.clear();
        }

        // Redrawn from scratch on every render: the set of blocks and their colours both change
        // as work is recorded, and twenty-four markers are far too few for reuse to be worth it.
        state.markers.forEach(m => m.setMap(null));
        state.markers.clear();
        state.info.close();

        const bounds = new google.maps.LatLngBounds();
        for (const block of blocks) {
            const position = { lat: block.lat, lng: block.lng };
            // google.maps.Marker is the classic marker. AdvancedMarkerElement is the successor but
            // needs a cloud-configured Map ID, which would make the screen depend on console setup
            // beyond an API key; a scaled circle symbol is exactly what this map wants anyway.
            const marker = new google.maps.Marker({
                map: state.map,
                position,
                title: block.tooltip,
                icon: {
                    path: google.maps.SymbolPath.CIRCLE,
                    scale: block.scale,
                    fillColor: block.colour,
                    fillOpacity: 0.85,
                    strokeColor: '#ffffff',
                    strokeWeight: block.id === options.selectedId ? 3 : 1.5
                },
                zIndex: block.id === options.selectedId ? 10 : 1
            });

            marker.addListener('click', () => {
                state.info.setContent(
                    `<div style="font:600 13px system-ui,sans-serif">${block.code}</div>`
                    + `<div style="font:12px system-ui,sans-serif;color:#546e7a">${block.tooltip}</div>`);
                state.info.open({ map: state.map, anchor: marker });
                state.dotNet?.invokeMethodAsync('OnMarkerSelected', block.id);
            });

            state.markers.set(block.id, marker);
            bounds.extend(position);
        }

        if (options.fitBounds !== false) {
            state.map.fitBounds(bounds, 48);
        }

        // A rejected key does not fail the script load: Google serves the API, then calls
        // gm_authFailure once it has checked the key and leaves a greyed-out, watermarked map
        // behind. Waiting for the first idle — or a moment, whichever comes first — is what turns
        // that into an honest "fall back to the plot" instead of a map nobody can use.
        await new Promise(resolve => {
            const done = google.maps.event.addListenerOnce(state.map, 'idle', resolve);
            setTimeout(() => { google.maps.event.removeListener(done); resolve(); }, 3000);
        });
        if (state.authFailed) {
            dispose();
            host.replaceChildren();     // otherwise Google's watermarked shell stays in the page
            return { ok: false, reason: 'Google rejected the configured Maps API key.' };
        }

        return { ok: true };
    }

    /** Centres the map on one block; used when a row in the side list is chosen. */
    function focus(blockId, zoom) {
        const marker = state.markers.get(blockId);
        if (!marker || !state.map) { return false; }
        state.map.panTo(marker.getPosition());
        if (zoom) { state.map.setZoom(zoom); }
        google.maps.event.trigger(marker, 'click');
        return true;
    }

    function dispose() {
        state.markers.forEach(m => m.setMap(null));
        state.markers.clear();
        state.info?.close();
        state.map = null;
        state.dotNet = null;
    }

    window.sgcBlockMap = { render, focus, dispose };
})();
