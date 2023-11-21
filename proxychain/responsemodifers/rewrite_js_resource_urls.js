// Overrides the global fetch and XMLHttpRequest open methods to modify the request URLs.
// Also overrides the attribute setter prototype to modify the request URLs
// fetch("/relative_script.js") -> fetch("http://localhost:8080/relative_script.js")
(() => {
   const blacklistedSchemes = [
    "ftp:",
    "mailto:",
    "tel:",
    "file:",
    "blob:",
    "javascript:",
    "about:",
    "magnet:",
    "ws:",
    "wss:",
   ];

   function rewriteURL(url) {
        const oldUrl = url 
        if (!url) return url
        // don't rewrite invalid URIs
        try { new URL(url) } catch { return url }

        // don't rewrite special URIs
        if (blacklistedSchemes.includes(url)) return url;

        // don't double rewrite
        const proxyOrigin = globalThis.window.location.origin;
        if (url.startsWith(proxyOrigin)) return url;
        if (url.startsWith(`/${proxyOrigin}`)) return url;
        if (url.startsWith(`/${origin}`)) return url;

        const origin = (new URL(decodeURIComponent(globalThis.window.location.pathname.substring(1)))).origin
        //console.log(`proxychain: origin: ${origin} // proxyOrigin: ${proxyOrigin} // original: ${oldUrl}`)

        if (url.startsWith("//")) {
            url = `/${origin}/${encodeURIComponent(url.substring(2))}`;
        } else if (url.startsWith("/")) {
            url = `/${origin}/${encodeURIComponent(url.substring(1))}`;
        } else if (url.startsWith(origin)) {
            url = `/${encodeURIComponent(url)}` 
        } else if (url.startsWith("http://") || url.startsWith("https://")) {
            url = `/${proxyOrigin}/${encodeURIComponent(url)}`;
        }
        console.log(`proxychain: rewrite JS URL: ${oldUrl} -> ${url}`)
        return url;
   };

   // monkey patch fetch
   const oldFetch = globalThis.fetch;
   globalThis.fetch = async (url, init) => {
        return oldFetch(rewriteURL(url), init)
   }

   // monkey patch xmlhttprequest
   const oldOpen = XMLHttpRequest.prototype.open;
   XMLHttpRequest.prototype.open = function(method, url, async = true, user = null, password = null) {
       return oldOpen.call(this, method, rewriteURL(url), async, user, password);
   };
   const oldSend = XMLHttpRequest.prototype.send;
   XMLHttpRequest.prototype.send = function(method, url) {
       return oldSend.call(this, method, rewriteURL(url));
   };

   // monkey patch service worker registration
   const oldRegister = ServiceWorkerContainer.prototype.register;
   ServiceWorkerContainer.prototype.register = function(scriptURL, options) {
        return oldRegister.call(this, rewriteURL(scriptURL), options)
   }

   // monkey patch URL.toString() method 
   const oldToString = URL.prototype.toString
   URL.prototype.toString = function() {
        let originalURL = oldToString.call(this)
        return rewriteURL(originalURL)
   }

   // monkey patch URL.toJSON() method 
   const oldToJson = URL.prototype.toString
   URL.prototype.toString = function() {
        let originalURL = oldToJson.call(this)
        return rewriteURL(originalURL)
   }

   // Monkey patch URL.href getter and setter
    const originalHrefDescriptor = Object.getOwnPropertyDescriptor(URL.prototype, 'href');
    Object.defineProperty(URL.prototype, 'href', {
        get: function() {
            let originalHref = originalHrefDescriptor.get.call(this);
            return rewriteURL(originalHref)
        },
        set: function(newValue) {
            originalHrefDescriptor.set.call(this, rewriteURL(newValue));
        }
    });

    // Monkey patch setter 
    const elements = [
        { tag: 'a', attribute: 'href' },
        { tag: 'img', attribute: 'src' },
        // { tag: 'img', attribute: 'srcset' }, // TODO: handle srcset
        { tag: 'script', attribute: 'src' },
        { tag: 'link', attribute: 'href' },
        { tag: 'link', attribute: 'icon' },
        { tag: 'iframe', attribute: 'src' },
        { tag: 'audio', attribute: 'src' },
        { tag: 'video', attribute: 'src' },
        { tag: 'source', attribute: 'src' },
        // { tag: 'source', attribute: 'srcset' }, // TODO: handle srcset
        { tag: 'embed', attribute: 'src' },
        { tag: 'embed', attribute: 'pluginspage' },
        { tag: 'html', attribute: 'manifest' },
        { tag: 'object', attribute: 'src' },
        { tag: 'input', attribute: 'src' },
        { tag: 'track', attribute: 'src' },
        { tag: 'form', attribute: 'action' },
        { tag: 'area', attribute: 'href' },
        { tag: 'base', attribute: 'href' },
        { tag: 'blockquote', attribute: 'cite' },
        { tag: 'del', attribute: 'cite' },
        { tag: 'ins', attribute: 'cite' },
        { tag: 'q', attribute: 'cite' },
        { tag: 'button', attribute: 'formaction' },
        { tag: 'input', attribute: 'formaction' },
        { tag: 'meta', attribute: 'content' },
        { tag: 'object', attribute: 'data' },
    ];

    elements.forEach(({ tag, attribute }) => {
        const proto = document.createElement(tag).constructor.prototype;
        const descriptor = Object.getOwnPropertyDescriptor(proto, attribute);
        if (descriptor && descriptor.set) {
            Object.defineProperty(proto, attribute, {
                ...descriptor,
                set(value) {
                    // calling rewriteURL will end up calling a setter for href,
                    // leading to a recusive loop and a Maximum call stack size exceeded
                    // error, so we guard against this with a local semaphore flag
                    const isRewritingSetKey = Symbol.for('isRewritingSet');
                    if (!this[isRewritingSetKey]) {
                        this[isRewritingSetKey] = true;
                        descriptor.set.call(this, rewriteURL(value));
                        //descriptor.set.call(this, value);
                        this[isRewritingSetKey] = false;
                    } else {
                        // Directly set the value without rewriting
                        descriptor.set.call(this, value);
                    }
                },
                get() {
                    const isRewritingGetKey = Symbol.for('isRewritingGet');
                    if (!this[isRewritingGetKey]) {
                        this[isRewritingGetKey] = true;
                        let oldURL = descriptor.get.call(this);
                        let newURL = rewriteURL(oldURL);
                        this[isRewritingGetKey] = false;
                        return newURL
                    } else {
                        return descriptor.get.call(this);
                    }
                }
            });
        }
    });
})();