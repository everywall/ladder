(() => {
  const UA = "{{USER_AGENT}}";

  // monkey-patch navigator.userAgent
  {
    const { get } = Object.getOwnPropertyDescriptor(
      Navigator.prototype,
      "userAgent",
    );
    Object.defineProperty(Navigator.prototype, "userAgent", {
      get: new Proxy(get, {
        apply() {
          return UA;
        },
      }),
    });
  }

  // monkey-patch navigator.appVersion
  {
    const { get } = Object.getOwnPropertyDescriptor(
      Navigator.prototype,
      "appVersion",
    );
    Object.defineProperty(Navigator.prototype, "appVersion", {
      get: new Proxy(get, {
        apply() {
          return UA.replace("Mozilla/", "");
        },
      }),
    });
  }

  // monkey-patch navigator.UserAgentData
  // Assuming UAParser is already loaded and available
  function spoofUserAgentData(uaString) {
    // Parse the user-agent string
    const parser = new UAParser(uaString);
    const parsedData = parser.getResult();

    // Extracted data
    const platform = parsedData.os.name;
    const browserName = parsedData.browser.name;
    const browserMajorVersion = parsedData.browser.major;
    const isMobile =
      /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(
        uaString,
      );

    // Overwrite navigator.userAgentData
    self.NavigatorUAData = self.NavigatorUAData || new class NavigatorUAData {
      brands = [{
        brand: browserName,
        version: browserMajorVersion,
      }];
      mobile = isMobile;
      platform = platform;
      toJSON() {
        return {
          brands: this.brands,
          mobile: this.mobile,
          platform: this.platform,
        };
      }
      getHighEntropyValues(hints) {
        const result = this.toJSON();
        // Add additional high entropy values based on hints
        // Modify these as per your requirements
        if (hints.includes("architecture")) {
          result.architecture = "x86";
        }
        if (hints.includes("bitness")) {
          result.bitness = "64";
        }
        if (hints.includes("model")) {
          result.model = "";
        }
        if (hints.includes("platformVersion")) {
          result.platformVersion = "10.0.0"; // Example value
        }
        if (hints.includes("uaFullVersion")) {
          result.uaFullVersion = browserMajorVersion;
        }
        if (hints.includes("fullVersionList")) {
          result.fullVersionList = this.brands;
        }
        return Promise.resolve(result);
      }
    }();

    // Apply the monkey patch
    Object.defineProperty(navigator, "userAgentData", {
      value: new self.NavigatorUAData(),
      writable: false,
    });
  }

  spoofUserAgentData(UA);
  // TODO: use hideMonkeyPatch to hide overrides
})();
