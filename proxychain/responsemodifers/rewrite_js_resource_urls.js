// Overrides the global fetch and XMLHttpRequest open methods to modify the request URLs.
// Also overrides the attribute setter prototype to modify the request URLs
// fetch("/relative_script.js") -> fetch("http://localhost:8080/relative_script.js")
(() => {
   function rewriteURL(url) {
        oldUrl = url 
        if (!url) return url

        proxyOrigin = globalThis.window.location.origin
        if (url.startsWith(proxyOrigin)) return url

        const origin = (new URL(decodeURI(globalThis.window.location.pathname.substring(1)))).origin
        if (url.startsWith("//")) {
            url = `/${origin}/${encodeURIComponent(url.substring(2))}`;
        } else if (url.startsWith("/")) {
            url = `/${origin}/${encodeURIComponent(url.substring(1))}`;
        } else if (url.startsWith("http://") || url.startsWith("https://")) {
            url = `/${origin}/${encodeURIComponent(url)}`;
        }
        console.log(`rewrite JS URL: ${oldUrl} -> ${url}`)
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

    // Monkey patch setter methods
    const elements = [
        { tag: 'a', attribute: 'href' },
        { tag: 'img', attribute: 'src' },
        { tag: 'script', attribute: 'src' },
        { tag: 'link', attribute: 'href' },
        { tag: 'iframe', attribute: 'src' },
        { tag: 'audio', attribute: 'src' },
        { tag: 'video', attribute: 'src' },
        { tag: 'source', attribute: 'src' },
        { tag: 'embed', attribute: 'src' },
        { tag: 'object', attribute: 'src' },
        { tag: 'input', attribute: 'src' },
        { tag: 'track', attribute: 'src' },
        { tag: 'form', attribute: 'action' },
    ];

    elements.forEach(({ tag, attribute }) => {
        const proto = document.createElement(tag).constructor.prototype;
        const descriptor = Object.getOwnPropertyDescriptor(proto, attribute);
        if (descriptor && descriptor.set) {
            Object.defineProperty(proto, attribute, {
                ...descriptor,
                set(value) {
                    return descriptor.set.call(this, rewriteURL(value));
                }
            });
        }
    });

})();