// Overrides the global fetch and XMLHttpRequest open methods to modify the request URLs.
// Also overrides the attribute setter prototype to modify the request URLs
// fetch("/relative_script.js") -> fetch("http://localhost:8080/relative_script.js")
(() => {
   function rewriteURL(url) {
        if (!url) return url
        if (url.startsWith(window.location.origin)) return url

        if (url.startsWith("//")) {
            url = `${window.location.origin}/${encodeURIComponent(url.substring(2))}`;
        } else if (url.startsWith("/")) {
            url = `${window.location.origin}/${encodeURIComponent(url.substring(1))}`;
        } else if (url.startsWith("http://") || url.startsWith("https://")) {
            url = `${window.location.origin}/${encodeURIComponent(url)}`;
        }
        return url;
   };

   // monkey patch fetch
   const oldFetch = globalThis.fetch ;
   globalThis.fetch = async (url, init) => {
        return oldFetch(rewriteURL(url), init)
   }

   // monkey patch xmlhttprequest
   const oldOpen = XMLHttpRequest.prototype.open;
   XMLHttpRequest.prototype.open = function(method, url, async = true, user = null, password = null) {
       return oldOpen.call(this, method, rewriteURL(url), async, user, password);
   };


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