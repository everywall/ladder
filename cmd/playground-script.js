// TODO: Untoggle related items that may be toggled (e.g. should only have one masquerade as bot toggled)
// TODO: remove tailwind play cdn script in head
// TODO: test functionality and POST requests

const modifierContainer = document.getElementById("modifierContainer");
const modalContainer = document.getElementById("modalContainer");
const modalBody = document.getElementById("modal-body");
const modalContent = document.getElementById("modal-content");
const modalSubmitButton = document.getElementById("modal-submit");
const modalClose = document.getElementById("modal-close");

// Rerun handleThemeChange() so style is applied to Ninja Keys
handleThemeChange();

// Add event listener to the iframe so it closes dropdown when clicked
closeDropdownOnClickWithinIframe();

function closeDropdownOnClickWithinIframe() {
  const iframe = document.getElementById("resultIframe");
  iframe.contentWindow.document.addEventListener(
    "click",
    () => {
      if (
        !document.getElementById("dropdown_panel").classList.contains("hidden")
      ) {
        toggleDropdown();
      }
    },
    true
  );
}

document.getElementById("inputForm").addEventListener("submit", function (e) {
  e.preventDefault();
  submitForm();
});

if (navigator.platform.includes("Mac")) {
  document.getElementById("ninjaKey").innerHTML = "⌘";
} else {
  document.getElementById("ninjaKey").innerHTML = "Ctrl";
}

function downloadYaml() {
  function parseYaml() {
    // TODO parse payload into yaml format
    return payload;
  }

  const yamlData = parseYaml();
  const blob = new Blob([yamlData], { type: "text/yaml;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  // TODO construct filename from hostname into naming scheme
  link.download = `name_of_report.yaml`;
  link.click();
  URL.revokeObjectURL(url);
}

function submitForm() {
  if (!document.getElementById("inputForm").checkValidity()) {
    return;
  }
  fetch("/playground/" + inputField.value, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  })
    .then((response) => response.text())
    .then((result) => {
      const resultIframe = parent.document.getElementById("resultIframe");
      resultIframe.contentDocument.open();
      resultIframe.contentDocument.write(result);
      closeDropdownOnClickWithinIframe();
      resultIframe.contentDocument.close();
    });
}

function getValues(...fields) {
  const focusTrap = trap(modalBody);
  let values = {};
  const inputs = [];
  const inputEventListeners = [];

  function closeModal() {
    focusTrap.destroy();
    modalBody.removeEventListener("keydown", handleEscapeKey);
    modalBody.removeEventListener("keydown", handleEnterKey);
    modalContainer.removeEventListener("click", handleClickOutside);
    modalSubmitButton.removeEventListener("click", closeModal);
    modalClose.removeEventListener("click", closeModal);
    inputEventListeners.forEach((listener, index) => {
      inputs[index].removeEventListener("input", listener);
    });
    inputEventListeners.length = 0;
    inputs.length = 0;
    values = {};
    modalContainer.classList.add("hidden");
    modalContent.innerHTML = "";
  }

  const handleClickOutside = (e) => {
    if (modalBody !== null && !modalBody.contains(e.target)) {
      closeModal();
    }
  };

  const handleEscapeKey = (e) => {
    if (e.key === "Escape") {
      closeModal();
    }
  };

  const handleEnterKey = (e) => {
    if (e.key === "Enter") {
      modalSubmitButton.click();
    }
  };

  document.getElementById("modal-title").innerHTML = fields[0];

  for (let i = 1; i < fields.length; i++) {
    const label = document.createElement("label");
    label.innerHTML = fields[i];
    label.setAttribute("for", `input-${i}`);
    let input;
    if (fields[i] === "js") {
      input = document.createElement("textarea");
      input.type = "textarea";
      input.classList.add("min-h-[200px]");
    } else {
      input = document.createElement("input");
      input.type = "text";
    }
    input.id = `input-${i}`;
    input.classList.add(
      "w-full",
      "text-sm",
      "leading-6",
      "text-slate-400",
      "rounded-md",
      "ring-1",
      "ring-slate-900/10",
      "shadow-sm",
      "py-1.5",
      "pl-2",
      "pr-3",
      "hover:ring-slate-300",
      "dark:bg-slate-800",
      "dark:highlight-white/5",
      "dark:hover:bg-slate-700"
    );
    modalContent.appendChild(label);
    modalContent.appendChild(input);

    const inputEventListener = (event) => {
      const fieldName = fields[i];
      values[fieldName] = event.target.value;
    };
    input.addEventListener("input", inputEventListener);
    inputEventListeners.push(inputEventListener);
    inputs.push(input);
  }

  modalContainer.classList.remove("hidden");
  document.getElementById("input-1").focus();

  return new Promise((resolve) => {
    modalBody.addEventListener("keydown", handleEscapeKey);
    modalBody.addEventListener("keydown", handleEnterKey);
    modalContainer.addEventListener("click", handleClickOutside);
    modalClose.addEventListener("click", () => {
      closeModal();
    });
    modalSubmitButton.addEventListener("click", () => {
      closeModal();
      resolve(values);
    });
  });
}

function clickHandler(pill) {
  toggleModifier("", pill.id);
  pill.removeEventListener("click", () => clickHandler(pill));
  pill.remove();
}

function toggleModifier(modifierType, modifierKey, values) {
  if (modifierType === "") {
    for (const parentKey in payload) {
      if (payload.hasOwnProperty(parentKey)) {
        const childKeys = Object.keys(payload[parentKey]);
        if (childKeys.includes(modifierKey)) {
          modifierType = payload[parentKey];
          break;
        }
      }
    }
  }
  if (typeof modifierType[modifierKey] === "boolean") {
    modifierType[modifierKey] = !modifierType[modifierKey];
  }

  if (typeof modifierType[modifierKey] === "object") {
    if (Object.values(modifierType[modifierKey]).some((v) => v === "")) {
      Object.assign(modifierType[modifierKey], values);
    } else {
      for (const key in modifierType[modifierKey]) {
        modifierType[modifierKey][key] = "";
      }
    }
  }

  const existingPill = document.getElementById(modifierKey);
  if (!existingPill) {
    const pill = document.createElement("span");
    pill.classList.add(
      "inline-flex",
      "items-center",
      "rounded-md",
      "bg-slate-100",
      "dark:bg-slate-800",
      "px-2",
      "py-1",
      "h-4",
      "text-xs",
      "font-medium",
      "border",
      "border-slate-400",
      "dark:border-slate-700",
      "cursor-pointer"
    );
    pill.id = modifierKey;
    pill.textContent = modifierKey;
    modifierContainer.appendChild(pill);
    pill.addEventListener("click", () => clickHandler(pill));
  } else {
    existingPill.removeEventListener("click", () => clickHandler(pill));
    existingPill.remove();
  }

  submitForm();
}

const ninja = document.querySelector("ninja-keys");
ninja.data = [
  {
    // REQUEST MODIFIERS
    id: "forwardRequestHeaders",
    title: "Forward request headers",
    section: "Request Modifiers",
    handler: () => {
      toggleModifier(payload.requestmodifierquery, "forwardrequestheaders");
    },
  },
  {
    id: "masqueradeAsTrustedBot",
    title: "Masquerade as ...",
    children: [
      "Google Bot",
      "Bing Bot",
      "Wayback Machine Bot",
      "Facebook Bot",
      "Yandex Bot",
      "Baidu Bot",
      "DuckDuckGo Bot",
      "Yahoo Bot",
    ],
    section: "Request Modifiers",
    handler: () => {
      ninja.open({ parent: "masqueradeAsTrustedBot" });
      return { keepOpen: true };
    },
  },
  {
    id: "masqueradeAsGoogleBot",
    title: "Google Bot",
    parent: "masqueradeAsTrustedBot",
    section: "Masquerade as ...",
    handler: () => {
      toggleModifier(payload.requestmodifierquery, "masqueradeasgooglebot");
    },
  },
  {
    id: "masqueradeAsBingBot",
    title: "Bing Bot",
    parent: "masqueradeAsTrustedBot",
    section: "Masquerade as ...",
    handler: () => {
      toggleModifier(payload.requestmodifierquery, "masqueradeasbingbot");
    },
  },
  {
    id: "masqueradeAsWaybackMachineBot",
    title: "Wayback Machine Bot",
    parent: "masqueradeAsTrustedBot",
    section: "Masquerade as ...",
    handler: () => {
      toggleModifier(
        payload.requestmodifierquery,
        "masqueradeaswaybackmachinebot"
      );
    },
  },
  {
    id: "masqueradeAsFacebookBot",
    title: "Facebook Bot",
    parent: "masqueradeAsTrustedBot",
    section: "Masquerade as ...",
    handler: () => {
      toggleModifier(payload.requestmodifierquery, "masqueradeasfacebookbot");
    },
  },
  {
    id: "masqueradeAsYandexBot",
    title: "Yandex Bot",
    parent: "masqueradeAsTrustedBot",
    section: "Masquerade as ...",
    handler: () => {
      toggleModifier(payload.requestmodifierquery, "masqueradeasyandexbot");
    },
  },
  {
    id: "masqueradeAsBaiduBot",
    title: "Baidu Bot",
    parent: "masqueradeAsTrustedBot",
    section: "Masquerade as ...",
    handler: () => {
      toggleModifier(payload.requestmodifierquery, "masqueradeabaidubot");
    },
  },
  {
    id: "masqueradeAsDuckDuckBot",
    title: "DuckDuckGo Bot",
    parent: "masqueradeAsTrustedBot",
    section: "Masquerade as ...",
    handler: () => {
      toggleModifier(payload.requestmodifierquery, "masqueradeasduckduckbot");
    },
  },
  {
    id: "masqueradeAsYahooBot",
    title: "Yahoo Bot",
    parent: "masqueradeAsTrustedBot",
    section: "Masquerade as ...",
    handler: () => {
      toggleModifier(payload.requestmodifierquery, "masqueradeasyahoobot");
    },
  },
  {
    id: "modifyOutgoingCookies",
    title: "Modify outgoing cookies...",
    children: [
      "Set outgoing cookie",
      "Set outgoing cookies",
      "Delete outgoing cookie",
      "Delete outgoing cookies",
      "Delete outgoing cookies except",
    ],
    section: "Request Modifiers",
    handler: () => {
      ninja.open({ parent: "modifyOutgoingCookies" });
      return { keepOpen: true };
    },
  },
  {
    id: "setOutgoingCookie",
    title: "Set outgoing cookie",
    parent: "modifyOutgoingCookies",
    section: "Modify outgoing cookies ...",
    handler: () => {
      getValues("Set outgoing cookie", "name", "val").then((values) => {
        if (Object.keys(values).length === 0) return;
        toggleModifier(
          payload.requestmodifierquery,
          "setoutgoingcookie",
          values
        );
      });
    },
  },
  {
    id: "setOutgoingCookies",
    title: "Set outgoing cookies",
    parent: "modifyOutgoingCookies",
    section: "Modify outgoing cookies ...",
    handler: () => {
      getValues("Set outgoing cookies", "cookies").then((values) => {
        if (Object.keys(values).length === 0) return;
        toggleModifier(
          payload.requestmodifierquery,
          "setoutgoingcookies",
          values
        );
      });
    },
  },
  {
    id: "deleteOutgoingCookie",
    title: "Delete outgoing cookie",
    parent: "modifyOutgoingCookies",
    section: "Modify outgoing cookies ...",
    handler: () => {
      getValues("Delete outgoing cookie", "name").then((values) => {
        if (Object.keys(values).length === 0) return;
        toggleModifier(
          payload.requestmodifierquery,
          "deleteoutgoingcookie",
          values
        );
      });
    },
  },
  {
    id: "deleteOutgoingCookies",
    title: "Delete outgoing cookies",
    parent: "modifyOutgoingCookies",
    section: "Modify outgoing cookies ...",
    handler: () => {
      toggleModifier(payload.requestmodifierquery, "deleteoutgoingcookies");
    },
  },
  {
    id: "deleteOutgoingCookiesExcept",
    title: "Delete outgoing cookies except",
    parent: "modifyOutgoingCookies",
    section: "Modify outgoing cookies ...",
    handler: () => {
      getValues("Set outgoing cookies except", "whitelist").then((values) => {
        if (Object.keys(values).length === 0) return;
        toggleModifier(
          payload.requestmodifierquery,
          "deleteoutgoingcookiesexcept",
          values
        );
      });
    },
  },
  {
    id: "modifyUrl",
    title: "Modify URL...",
    children: [
      "Modify domain with regex",
      "Modify path with regex",
      "Modify query params",
    ],
    section: "Request Modifiers",
    handler: () => {
      ninja.open({ parent: "modifyUrl" });
      return { keepOpen: true };
    },
  },
  {
    id: "modifyDomainWithRegex",
    title: "Modify domain with regex",
    parent: "modifyUrl",
    section: "Modify URL ...",
    handler: () => {
      getValues("Modify domain with regex", "match", "replacement").then(
        (values) => {
          if (Object.keys(values).length === 0) return;
          toggleModifier(
            payload.requestmodifierquery,
            "modifydomainwithregex",
            values
          );
        }
      );
    },
  },
  {
    id: "modifyPathWithRegex",
    title: "Modify path with regex",
    parent: "modifyUrl",
    section: "Modify URL ...",
    handler: () => {
      getValues("Modify path with regex", "match", "replacement").then(
        (values) => {
          if (Object.keys(values).length === 0) return;
          toggleModifier(
            payload.requestmodifierquery,
            "modifypathwithregex",
            values
          );
        }
      );
    },
  },
  {
    id: "modifyQueryParams",
    title: "Modify query params",
    parent: "modifyUrl",
    section: "Modify URL ...",
    handler: () => {
      getValues("Modify query params", "key", "value").then((values) => {
        if (Object.keys(values).length === 0) return;
        toggleModifier(
          payload.requestmodifierquery,
          "modifyqueryparams",
          values
        );
      });
    },
  },
  {
    id: "modifyRequestHeaders",
    title: "Modify request headers...",
    children: ["Set request header", "Delete request header"],
    section: "Request Modifiers",
    handler: () => {
      ninja.open({ parent: "modifyRequestHeaders" });
      return { keepOpen: true };
    },
  },
  {
    id: "setRequestHeader",
    title: "Set request header",
    parent: "modifyRequestHeaders",
    section: "Modify request headers ...",
    handler: () => {
      getValues("Set request header", "name", "val").then((values) => {
        if (Object.keys(values).length === 0) return;
        toggleModifier(
          payload.requestmodifierquery,
          "setrequestheader",
          values
        );
      });
    },
  },
  {
    id: "deleteRequestHeader",
    title: "Delete request header",
    parent: "modifyRequestHeaders",
    section: "Modify request headers ...",
    handler: () => {
      getValues("Delete request header", "name").then((values) => {
        if (Object.keys(values).length === 0) return;
        toggleModifier(
          payload.requestmodifierquery,
          "deleterequestheader",
          values
        );
      });
    },
  },
  {
    id: "requestArchived",
    title: "Request archived version from...",
    children: ["Archive.is", "Google Cache", "Wayback Machine"],
    section: "Request Modifiers",
    handler: () => {
      ninja.open({ parent: "requestArchived" });
      return { keepOpen: true };
    },
  },
  {
    id: "requestArchiveIs",
    title: "Achive.is",
    parent: "requestArchived",
    section: "Reqest archived version from...",
    handler: () => {
      toggleModifier(payload.requestmodifierquery, "requestarchiveis");
    },
  },
  {
    id: "requestGoogleCache",
    title: "Google Cache",
    parent: "requestArchived",
    section: "Reqest archived version from...",
    handler: () => {
      toggleModifier(payload.requestmodifierquery, "requestgooglecache");
    },
  },
  {
    id: "requestWaybackMachine",
    title: "Wayback Machine",
    parent: "requestArchived",
    section: "Reqest archived version from...",
    handler: () => {
      toggleModifier(payload.requestmodifierquery, "requestwaybackmachine");
    },
  },
  {
    id: "resolveWithGoogleDoH",
    title: "Resolve with Google's DNS over HTTPs service",
    section: "Request Modifiers",
    handler: () => {
      toggleModifier(payload.requestmodifierquery, "resolvewithgoogledoh");
    },
  },
  {
    id: "spoofRequest",
    title: "Spoof or hide...",
    children: ["Archive.is", "Google Cache", "Wayback Machine"],
    section: "Request Modifiers",
    handler: () => {
      ninja.open({ parent: "spoofRequest" });
      return { keepOpen: true };
    },
  },
  {
    id: "spoofOrigin",
    title: "Spoof origin",
    parent: "spoofRequest",
    section: "Spoof or hide...",
    handler: () => {
      getValues("Spoof origin", "url").then((values) => {
        if (Object.keys(values).length === 0) return;
        toggleModifier(payload.requestmodifierquery, "spooforigin", values);
      });
    },
  },
  {
    id: "hideOrigin",
    title: "Hide origin",
    parent: "spoofRequest",
    section: "Spoof or hide...",
    handler: () => {
      toggleModifier(payload.requestmodifierquery, "hideorigin");
    },
  },
  {
    id: "spoofReferrer",
    title: "Spoof referrer",
    parent: "spoofRequest",
    section: "Spoof or hide...",
    handler: () => {
      getValues("Spoof referrer", "url").then((values) => {
        if (Object.keys(values).length === 0) return;
        toggleModifier(payload.requestmodifierquery, "spoofreferrer", values);
      });
    },
  },
  {
    id: "hideReferrer",
    title: "Hide referrer",
    parent: "spoofRequest",
    section: "Spoof or hide...",
    handler: () => {
      toggleModifier(payload.requestmodifierquery, "hidereferrer");
    },
  },
  {
    id: "spoofReferrerFromBaiduSearch",
    title: "Spoof referrer from Baidu search",
    parent: "spoofRequest",
    section: "Spoof or hide...",
    handler: () => {
      toggleModifier(
        payload.requestmodifierquery,
        "spoofreferrerfrombaidusearch"
      );
    },
  },
  {
    id: "spoofReferrerFromBingSearch",
    title: "Spoof referrer from Bing search",
    parent: "spoofRequest",
    section: "Spoof or hide...",
    handler: () => {
      toggleModifier(
        payload.requestmodifierquery,
        "spoofreferrerfrombingsearch"
      );
    },
  },
  {
    id: "spoofReferrerFromGoogleSearch",
    title: "Spoof referrer from Google search",
    parent: "spoofRequest",
    section: "Spoof or hide...",
    handler: () => {
      toggleModifier(
        payload.requestmodifierquery,
        "spoofreferrerfromgooglesearch"
      );
    },
  },
  {
    id: "spoofReferrerFromLinkedinPost",
    title: "Spoof referrer from Linkedin post",
    parent: "spoofRequest",
    section: "Spoof or hide...",
    handler: () => {
      toggleModifier(
        payload.requestmodifierquery,
        "spoofreferrerfromlinkedinpost"
      );
    },
  },
  {
    id: "spoofReferrerFromNaverSearch",
    title: "Spoof referrer from Naver search",
    parent: "spoofRequest",
    section: "Spoof or hide...",
    handler: () => {
      toggleModifier(
        payload.requestmodifierquery,
        "spoofreferrerfromnaversearch"
      );
    },
  },
  {
    id: "spoofReferrerFromPinterestPost",
    title: "Spoof referrer from Pinterest post",
    parent: "spoofRequest",
    section: "Spoof or hide...",
    handler: () => {
      toggleModifier(
        payload.requestmodifierquery,
        "spoofreferrerfrompinterestpost"
      );
    },
  },
  {
    id: "spoofReferrerFromQQPost",
    title: "Spoof referrer from QQ post",
    parent: "spoofRequest",
    section: "Spoof or hide...",
    handler: () => {
      toggleModifier(payload.requestmodifierquery, "spoofreferrerfromqqpost");
    },
  },
  {
    id: "spoofReferrerFromRedditPost",
    title: "Spoof referrer from Reddit post",
    parent: "spoofRequest",
    section: "Spoof or hide...",
    handler: () => {
      toggleModifier(
        payload.requestmodifierquery,
        "spoofreferrerfromredditpost"
      );
    },
  },
  {
    id: "spoofReferrerFromTumblrPost",
    title: "Spoof referrer from Tumblr post",
    parent: "spoofRequest",
    section: "Spoof or hide...",
    handler: () => {
      toggleModifier(
        payload.requestmodifierquery,
        "spoofreferrerfromtumblrpost"
      );
    },
  },
  {
    id: "spoofReferrerFromTwitterPost",
    title: "Spoof referrer from Twitter post",
    parent: "spoofRequest",
    section: "Spoof or hide...",
    handler: () => {
      toggleModifier(
        payload.requestmodifierquery,
        "spoofreferrerfromtwitterpost"
      );
    },
  },
  {
    id: "spoofReferrerFromKontaktePost",
    title: "Spoof referrer from Kontakte post",
    parent: "spoofRequest",
    section: "Spoof or hide...",
    handler: () => {
      toggleModifier(
        payload.requestmodifierquery,
        "spoofreferrerfromkontaktelepost"
      );
    },
  },
  {
    id: "spoofReferrerFromWeiboPost",
    title: "Spoof referrer from Weibo post",
    parent: "spoofRequest",
    section: "Spoof or hide...",
    handler: () => {
      toggleModifier(
        payload.requestmodifierquery,
        "spoofreferrerfromweibopost"
      );
    },
  },
  {
    id: "spoofUserAgent",
    title: "Spoof user agent",
    parent: "spoofRequest",
    section: "Spoof or hide...",
    handler: () => {
      getValues("Spoof user agent", "ua").then((values) => {
        if (Object.keys(values).length === 0) return;
        toggleModifier(payload.requestmodifierquery, "spoofuseragent", values);
      });
    },
  },
  {
    id: "spoofXForwardedFor",
    title: "Spoof X-Forwarded-For",
    parent: "spoofRequest",
    section: "Spoof or hide...",
    handler: () => {
      getValues("Spoof X-Forwarded-For", "ip").then((values) => {
        if (Object.keys(values).length === 0) return;
        toggleModifier(
          payload.requestmodifierquery,
          "spoofxforwardedfor",
          values
        );
      });
    },
  },
  // RESPONSE MODIFIERS
  {
    id: "APIContent",
    title: "Fetch JSON API return of content",
    section: "Response Modifiers",
    handler: () => {
      toggleModifier(payload.responsemodifierquery, "apicontent");
    },
  },
  {
    id: "blockElementRemoval",
    title: "Block CSS element removal",
    section: "Response Modifiers",
    handler: () => {
      getValues("Block CSS element removal", "cssSelector").then((values) => {
        if (Object.keys(values).length === 0) return;
        toggleModifier(
          payload.responsemodifierquery,
          "blockelementremoval",
          values
        );
      });
    },
  },
  {
    id: "bypassCors",
    title: "Bypass CORS",
    section: "Response Modifiers",
    handler: () => {
      toggleModifier(payload.responsemodifierquery, "bypasscors");
    },
  },
  {
    id: "bypassContentSecurityPolicy",
    title: "Bypass Content Security Policy",
    section: "Response Modifiers",
    handler: () => {
      getValues("Bypass Content Security Policy", "csp").then((values) => {
        if (Object.keys(values).length === 0) return;
        toggleModifier(
          payload.responsemodifierquery,
          "bypasscontentsecuritypolicy",
          values
        );
      });
    },
  },
  {
    id: "forwardResponseHeaders",
    title: "Forward response headers",
    section: "Response Modifiers",
    handler: () => {
      toggleModifier(payload.responsemodifierquery, "forwardresponseheaders");
    },
  },
  {
    id: "generateReadableOutline",
    title: "Generate readable outline",
    section: "Response Modifiers",
    handler: () => {
      toggleModifier(payload.responsemodifierquery, "generatereadableoutline");
    },
  },
  {
    id: "injectScript",
    title: "Inject script...",
    children: [
      "Inject script before DOM content loaded",
      "Inject script after DOM content loaded",
      "Inject script after DOM idle",
    ],
    section: "Response Modifiers",
    handler: () => {
      ninja.open({ parent: "injectScript" });
      return { keepOpen: true };
    },
  },
  {
    id: "injectScriptBeforeDOMContentLoaded",
    title: "Inject script before DOM content loaded",
    parent: "injectScript",
    section: "Inject script...",
    handler: () => {
      getValues("Inject script before DOM content loaded", "js").then(
        (values) => {
          if (Object.keys(values).length === 0) return;
          toggleModifier(
            payload.responsemodifierquery,
            "injectscriptbeforedomcontentloaded",
            values
          );
        }
      );
    },
  },
  {
    id: "injectScriptAfterDOMContentLoaded",
    title: "Inject script after DOM content loaded",
    parent: "injectScript",
    section: "Inject script...",
    handler: () => {
      getValues("Inject script after DOM content loaded", "js").then(
        (values) => {
          if (Object.keys(values).length === 0) return;
          toggleModifier(
            payload.responsemodifierquery,
            "injectscriptafterdomcontentloaded",
            values
          );
        }
      );
    },
  },
  {
    id: "injectScriptAfterDOMIdle",
    title: "Inject script after DOM idle",
    parent: "injectScript",
    section: "Inject script...",
    handler: () => {
      getValues("Inject script after DOM idle", "js").then((values) => {
        if (Object.keys(values).length === 0) return;
        toggleModifier(
          payload.responsemodifierquery,
          "injectscriptafterdomidle",
          values
        );
      });
    },
  },
  {
    id: "modifyIncomingCookies",
    title: "Modify incoming cookies...",
    children: [
      "Delete incoming cookies",
      "Delete incoming cookies except",
      "Set incoming cookies",
      "Set incoming cookie",
    ],
    section: "Response Modifiers",
    handler: () => {
      ninja.open({ parent: "modifyIncomingCookies" });
      return { keepOpen: true };
    },
  },
  {
    id: "deleteIncomingCookies",
    title: "Delete incoming cookies",
    parent: "modifyIncomingCookies",
    section: "Modify incoming cookies...",
    handler: () => {
      toggleModifier(payload.responsemodifierquery, "deleteincomingcookies");
    },
  },
  {
    id: "deleteIncomingCookiesExcept",
    title: "Delete incoming cookies except",
    parent: "modifyIncomingCookies",
    section: "Modify incoming cookies...",
    handler: () => {
      getValues("Delete incoming cookies except", "whitelist").then(
        (values) => {
          if (Object.keys(values).length === 0) return;
          toggleModifier(
            payload.responsemodifierquery,
            "deleteincomingcookiesexcept",
            values
          );
        }
      );
    },
  },
  {
    id: "setIncomingCookies",
    title: "Set incoming cookies",
    parent: "modifyIncomingCookies",
    section: "Modify incoming cookies...",
    handler: () => {
      getValues("Set incoming cookies", "cookies").then((values) => {
        if (Object.keys(values).length === 0) return;
        toggleModifier(
          payload.responsemodifierquery,
          "setincomingcookies",
          values
        );
      });
    },
  },
  {
    id: "setIncomingCookie",
    title: "Set incoming cookie",
    parent: "modifyIncomingCookies",
    section: "Modify incoming cookies...",
    handler: () => {
      getValues("Set incoming cookie", "name", "val").then((values) => {
        if (Object.keys(values).length === 0) return;
        toggleModifier(
          payload.responsemodifierquery,
          "setincomingcookie",
          values
        );
      });
    },
  },
  {
    id: "modifyResponseHeaders",
    title: "Modify response headers...",
    children: ["Set response header", "Delete response header"],
    section: "Response Modifiers",
    handler: () => {
      ninja.open({ parent: "modifyResponseHeaders" });
      return { keepOpen: true };
    },
  },
  {
    id: "setResponseHeader",
    title: "Set response header",
    parent: "modifyResponseHeaders",
    section: "Modify response headers...",
    handler: () => {
      getValues("Set response header", "key", "value").then((values) => {
        if (Object.keys(values).length === 0) return;
        toggleModifier(
          payload.responsemodifierquery,
          "setresponseheader",
          values
        );
      });
    },
  },
  {
    id: "deleteResponseHeader",
    title: "Delete response header",
    parent: "modifyResponseHeaders",
    section: "Modify response headers...",
    handler: () => {
      getValues("Delete response header", "key").then((values) => {
        if (Object.keys(values).length === 0) return;
        toggleModifier(
          payload.responsemodifierquery,
          "deleteresponseheader",
          values
        );
      });
    },
  },
  {
    id: "patchDynamicResourceUrls",
    title: "Patch dynamic resource urls",
    section: "Response Modifiers",
    handler: () => {
      toggleModifier(payload.responsemodifierquery, "patchdynamicresourceurls");
    },
  },
  {
    id: "patchGoogleAnalytics",
    title: "Patch Google Analytics",
    section: "Response Modifiers",
    handler: () => {
      toggleModifier(payload.responsemodifierquery, "patchgoogleanalytics");
    },
  },
  {
    id: "patchTrackerscripts",
    title: "Patch tracker scripts",
    section: "Response Modifiers",
    handler: () => {
      toggleModifier(payload.responsemodifierquery, "patchtrackerscripts");
    },
  },
  {
    id: "rewriteHtmlResourceUrls",
    title: "Rewrite HTML resource urls",
    section: "Response Modifiers",
    handler: () => {
      toggleModifier(payload.responsemodifierquery, "rewritehtmlresourceurls");
    },
  },
];
document.getElementById("btnNinja").addEventListener("click", () => {
  ninja.open();
});

let payload = {
  requestmodifierquery: {
    forwardrequestheaders: false,
    masqueradeasgooglebot: false,
    masqueradeasbingbot: false,
    masqueradeaswaybackmachinebot: false,
    masqueradeasfacebookbot: false,
    masqueradeasyandexbot: false,
    masqueradeasbaidubot: false,
    masqueradeasduckduckbot: false,
    masqueradeasyahoobot: false,
    modifydomainwithregex: {
      match: "",
      replacement: "",
    },
    setoutgoingcookie: {
      name: "",
      val: "",
    },
    setoutgoingcookies: {
      cookies: "",
    },
    deleteoutgoingcookie: {
      name: "",
    },
    deleteoutgoingcookies: false,
    deleteoutgoingcookiesexcept: {
      whitelist: "",
    },
    modifypathwithregex: {
      match: "",
      replacement: "",
    },
    modifyqueryparams: {
      key: "",
      value: "",
    },
    setrequestheader: {
      name: "",
      val: "",
    },
    deleterequestheader: {
      name: "",
    },
    requestarchiveis: false,
    requestgooglecache: false,
    requestwaybackmachine: false,
    resolvewithgoogledoh: false,
    spooforigin: {
      url: "",
    },
    hideorigin: false,
    spoofreferrer: {
      url: "",
    },
    hidereferrer: false,
    spoofreferrerfrombaidusearch: false,
    spoofreferrerfrombingsearch: false,
    spoofreferrerfromgooglesearch: false,
    spoofreferrerfromlinkedinpost: false,
    spoofreferrerfromnaversearch: false,
    spoofreferrerfrompinterestpost: false,
    spoofreferrerfromqqpost: false,
    spoofreferrerfromredditpost: false,
    spoofreferrerfromtumblrpost: false,
    spoofreferrerfromtwitterpost: false,
    spoofreferrerfromvkontaktepost: false,
    spoofreferrerfromweibopost: false,
    spoofuseragent: {
      ua: "",
    },
    spoofxforwardedfor: {
      ip: "",
    },
  },
  responsemodifierquery: {
    apicontent: false,
    blockelementremoval: {
      cssSelector: "",
    },
    bypasscors: false,
    bypasscontentsecuritypolicy: false,
    setcontentsecuritypolicy: {
      csp: "",
    },
    forwardresponseheaders: false,
    generatereadableoutline: false,
    injectscriptbeforedomcontentloaded: {
      js: "",
    },
    injectscriptafterdomcontentloaded: {
      js: "",
    },
    injectscriptafterdomidle: {
      js: "",
    },
    deleteincomingcookies: false,
    deleteincomingcookiesexcept: {
      whitelist: "",
    },
    setincomingcookies: {
      cookies: "",
    },
    setincomingcookie: {
      name: "",
      val: "",
    },
    setresponseheader: {
      key: "",
      value: "",
    },
    deleteresponseheader: {
      key: "",
    },
    patchdynamicresourceurls: false,
    patchgoogleanalytics: false,
    patchtrackerscripts: false,
    rewritehtmlresourceurls: false,
  },
};
