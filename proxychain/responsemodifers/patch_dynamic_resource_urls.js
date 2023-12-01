// Overrides the global fetch and XMLHttpRequest open methods to modify the request URLs.
// Also overrides the attribute setter prototype to modify the request URLs
// fetch("/relative_script.js") -> fetch("http://localhost:8080/relative_script.js")
(() => {
    // ============== PARAMS ===========================
    // if the original request was: http://localhost:8080/http://proxiedsite.com/foo/bar
    // proxyOrigin is http://localhost:8080
    const proxyOrigin = "{{PROXY_ORIGIN}}";
    //const proxyOrigin = globalThis.window.location.origin;

    // if the original request was: http://localhost:8080/http://proxiedsite.com/foo/bar
    // origin is http://proxiedsite.com
    const origin = "{{ORIGIN}}";
    //const origin = (new URL(decodeURIComponent(globalThis.window.location.pathname.substring(1)))).origin
    // ============== END PARAMS ======================

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
        if (!url) return url;

        // fetch url might be string, url, or request object
        // handle all three by downcasting to string
        const isStr = typeof url === "string";
        if (!isStr) {
            x = String(url);
            if (x == "[object Request]") {
                url = url.url;
            } else {
                url = String(url);
            }
        }

        const oldUrl = url;

        // don't rewrite special URIs
        if (blacklistedSchemes.includes(url)) return url;

        // don't rewrite invalid URIs
        try {
            new URL(url, origin);
        } catch {
            return url;
        }

        // don't double rewrite
        if (url.startsWith(`${proxyOrigin}/http://`)) return url;
        if (url.startsWith(`${proxyOrigin}/https://`)) return url;
        if (url.startsWith(`/${proxyOrigin}`)) return url;
        if (url.startsWith(`/${origin}`)) return url;
        if (url.startsWith(`/http://`)) return url;
        if (url.startsWith(`/https://`)) return url;
        if (url.startsWith(`/http%3A%2F%2F`)) return url;
        if (url.startsWith(`/https%3A%2F%2F`)) return url;
        if (url.startsWith(`/%2Fhttp`)) return url;

        //console.log(`proxychain: origin: ${origin} // proxyOrigin: ${proxyOrigin} // original: ${oldUrl}`)

        if (url.startsWith("//")) {
            url = `/${origin}/${encodeURIComponent(url.substring(2))}`;
        } else if (url.startsWith("/")) {
            url = `/${origin}/${encodeURIComponent(url.substring(1))}`;
        } else if (
            url.startsWith(proxyOrigin) && !url.startsWith(`${proxyOrigin}/http`)
        ) {
            // edge case where client js uses current url host to write an absolute path
            url = "".replace(proxyOrigin, `${proxyOrigin}/${origin}`);
        } else if (url.startsWith(origin)) {
            url = `/${encodeURIComponent(url)}`;
        } else if (url.startsWith("http://") || url.startsWith("https://")) {
            url = `/${proxyOrigin}/${encodeURIComponent(url)}`;
        }
        console.log(`proxychain: rewrite JS URL: ${oldUrl} -> ${url}`);
        return url;
    }

    /*
                // sometimes anti-bot protections like cloudflare or akamai bot manager check if JS is hooked
                function hideMonkeyPatch(objectOrName, method, originalToString) {
                    let obj;
                    let isGlobalFunction = false;
  
                    if (typeof objectOrName === "string") {
                        obj = globalThis[objectOrName];
                        isGlobalFunction = (typeof obj === "function") &&
                            (method === objectOrName);
                    } else {
                        obj = objectOrName;
                    }
  
                    if (isGlobalFunction) {
                        const originalFunction = obj;
                        globalThis[objectOrName] = function(...args) {
                            return originalFunction.apply(this, args);
                        };
                        globalThis[objectOrName].toString = () => originalToString;
                    } else if (obj && typeof obj[method] === "function") {
                        const originalMethod = obj[method];
                        obj[method] = function(...args) {
                            return originalMethod.apply(this, args);
                        };
                        obj[method].toString = () => originalToString;
                    } else {
                        console.warn(
                            `proxychain: cannot hide monkey patch: ${method} is not a function on the provided object.`,
                        );
                    }
                }
                */
    function hideMonkeyPatch(objectOrName, method, originalToString) {
        return;
    }

    // monkey patch fetch
    const oldFetch = fetch;
    fetch = async (url, init) => {
        return oldFetch(rewriteURL(url), init);
    };
    hideMonkeyPatch("fetch", "fetch", "function fetch() { [native code] }");

    // monkey patch xmlhttprequest
    const oldOpen = XMLHttpRequest.prototype.open;
    XMLHttpRequest.prototype.open = function(
        method,
        url,
        async = true,
        user = null,
        password = null,
    ) {
        return oldOpen.call(this, method, rewriteURL(url), async, user, password);
    };
    hideMonkeyPatch(
        XMLHttpRequest.prototype,
        "open",
        'function(){if("function"==typeof eo)return eo.apply(this,arguments)}',
    );

    const oldSend = XMLHttpRequest.prototype.send;
    XMLHttpRequest.prototype.send = function(method, url) {
        return oldSend.call(this, method, rewriteURL(url));
    };
    hideMonkeyPatch(
        XMLHttpRequest.prototype,
        "send",
        'function(){if("function"==typeof eo)return eo.apply(this,arguments)}',
    );

    // monkey patch service worker registration
    const oldRegister = ServiceWorkerContainer.prototype.register;
    ServiceWorkerContainer.prototype.register = function(scriptURL, options) {
        return oldRegister.call(this, rewriteURL(scriptURL), options);
    };
    hideMonkeyPatch(
        ServiceWorkerContainer.prototype,
        "register",
        "function register() { [native code] }",
    );

    // monkey patch URL.toString() method
    const oldToString = URL.prototype.toString;
    URL.prototype.toString = function() {
        let originalURL = oldToString.call(this);
        return rewriteURL(originalURL);
    };
    hideMonkeyPatch(
        URL.prototype,
        "toString",
        "function toString() { [native code] }",
    );

    // monkey patch URL.toJSON() method
    const oldToJson = URL.prototype.toString;
    URL.prototype.toString = function() {
        let originalURL = oldToJson.call(this);
        return rewriteURL(originalURL);
    };
    hideMonkeyPatch(
        URL.prototype,
        "toString",
        "function toJSON() { [native code] }",
    );

    // Monkey patch URL.href getter and setter
    const originalHrefDescriptor = Object.getOwnPropertyDescriptor(
        URL.prototype,
        "href",
    );
    Object.defineProperty(URL.prototype, "href", {
        get: function() {
            let originalHref = originalHrefDescriptor.get.call(this);
            return rewriteURL(originalHref);
        },
        set: function(newValue) {
            originalHrefDescriptor.set.call(this, rewriteURL(newValue));
        },
    });

    // TODO: do one more pass of this by manually traversing the DOM
    // AFTER all the JS and page has loaded just in case

    // Monkey patch setter
    const elements = [
        { tag: "a", attribute: "href" },
        { tag: "img", attribute: "src" },
        // { tag: 'img', attribute: 'srcset' }, // TODO: handle srcset
        { tag: "script", attribute: "src" },
        { tag: "link", attribute: "href" },
        { tag: "link", attribute: "icon" },
        { tag: "iframe", attribute: "src" },
        { tag: "audio", attribute: "src" },
        { tag: "video", attribute: "src" },
        { tag: "source", attribute: "src" },
        // { tag: 'source', attribute: 'srcset' }, // TODO: handle srcset
        { tag: "embed", attribute: "src" },
        { tag: "embed", attribute: "pluginspage" },
        { tag: "html", attribute: "manifest" },
        { tag: "object", attribute: "src" },
        { tag: "input", attribute: "src" },
        { tag: "track", attribute: "src" },
        { tag: "form", attribute: "action" },
        { tag: "area", attribute: "href" },
        { tag: "base", attribute: "href" },
        { tag: "blockquote", attribute: "cite" },
        { tag: "del", attribute: "cite" },
        { tag: "ins", attribute: "cite" },
        { tag: "q", attribute: "cite" },
        { tag: "button", attribute: "formaction" },
        { tag: "input", attribute: "formaction" },
        { tag: "meta", attribute: "content" },
        { tag: "object", attribute: "data" },
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
                    const isRewritingSetKey = Symbol.for("isRewritingSet");
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
                    const isRewritingGetKey = Symbol.for("isRewritingGet");
                    if (!this[isRewritingGetKey]) {
                        this[isRewritingGetKey] = true;
                        let oldURL = descriptor.get.call(this);
                        let newURL = rewriteURL(oldURL);
                        this[isRewritingGetKey] = false;
                        return newURL;
                    } else {
                        return descriptor.get.call(this);
                    }
                },
            });
        }
    });

    // sometimes, libraries will set the Element.innerHTML or Element.outerHTML directly with a string instead of setters.
    // in this case, we intercept it, create a fake DOM, parse it and then rewrite all attributes that could
    // contain a URL. Then we return the replacement innerHTML/outerHTML with redirected links.
    function rewriteInnerHTML(html, elements) {
        const isRewritingHTMLKey = Symbol.for("isRewritingHTML");

        // Check if already processing
        if (document[isRewritingHTMLKey]) {
            return html;
        }

        const tempContainer = document.createElement("div");
        document[isRewritingHTMLKey] = true;

        try {
            tempContainer.innerHTML = html;

            // Create a map for quick lookup
            const elementsMap = new Map(elements.map((e) => [e.tag, e.attribute]));

            // Loop-based DOM traversal
            const nodes = [...tempContainer.querySelectorAll("*")];
            for (const node of nodes) {
                const attribute = elementsMap.get(node.tagName.toLowerCase());
                if (attribute && node.hasAttribute(attribute)) {
                    const originalUrl = node.getAttribute(attribute);
                    const rewrittenUrl = rewriteURL(originalUrl);
                    node.setAttribute(attribute, rewrittenUrl);
                }
            }

            return tempContainer.innerHTML;
        } finally {
            // Clear the flag
            document[isRewritingHTMLKey] = false;
        }
    }

    // Store original setters
    const originalSetters = {};

    ["innerHTML", "outerHTML"].forEach((property) => {
        const descriptor = Object.getOwnPropertyDescriptor(
            Element.prototype,
            property,
        );
        if (descriptor && descriptor.set) {
            originalSetters[property] = descriptor.set;

            Object.defineProperty(Element.prototype, property, {
                ...descriptor,
                set(value) {
                    const isRewritingHTMLKey = Symbol.for("isRewritingHTML");
                    if (!this[isRewritingHTMLKey]) {
                        this[isRewritingHTMLKey] = true;
                        try {
                            // Use custom logic
                            descriptor.set.call(this, rewriteInnerHTML(value, elements));
                        } finally {
                            this[isRewritingHTMLKey] = false;
                        }
                    } else {
                        // Use original setter in recursive call
                        originalSetters[property].call(this, value);
                    }
                },
            });
        }
    });
})();
