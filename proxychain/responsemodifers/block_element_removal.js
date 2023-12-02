/**
 * Monitors and restores specific DOM elements if they are removed.
 *
 * This self-invoking function creates a MutationObserver to watch for removal of elements matching
 * "{{CSS_SELECTOR}}". If such an element is removed, it logs the event and attempts to restore the
 * element after a 50ms delay. The restored element is reinserted at its original location or prepended
 * to the document body if the original location is unavailable.
 */
(function() {
  function handleMutation(mutationList) {
    for (const mutation of mutationList) {
      if (mutation.type === "childList") {
        for (const node of Array.from(mutation.removedNodes)) {
          if (node.outerHTML && node.querySelector("{{CSS_SELECTOR}}")) {
            console.log(
              "proxychain: prevented removal of element containing 'article-content'",
            );
            console.log(node.outerHTML);
            setTimeout(() => {
              let e = document.querySelector("{{CSS_SELECTOR}}");
              if (e != null) {
                e.replaceWith(node);
              } else {
                document.body.prepend(node);
              }
            }, 50);
          }
        }
      }
    }
  }

  const observer = new MutationObserver(handleMutation);
  observer.observe(document, { childList: true, subtree: true });
})();
