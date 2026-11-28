# Changelog

## [v0.1.1](https://github.com/ostendorf-cloud/ostendorf.tech/compare/v0.1.0...v0.1.1) (2026-11-28)

### Bug Fixes

- **ci:** use the client id for the release app token ([#182](https://github.com/ostendorf-cloud/ostendorf.tech/pull/182))
- **deps:** update govue to v0.12.0 ([#173](https://github.com/ostendorf-cloud/ostendorf.tech/pull/173))
- **deps:** update govue to v0.11.0 ([#168](https://github.com/ostendorf-cloud/ostendorf.tech/pull/168))

## v0.1.0 (2026-11-25)

### Features

- **web:** centre the header navigation ([#167](https://github.com/ostendorf-cloud/ostendorf.tech/pull/167))
- **web:** show the cycle theme button and one language link always ([#164](https://github.com/ostendorf-cloud/ostendorf.tech/pull/164))
- split into frontend, API and feed images on govue v0.10.0 ([#153](https://github.com/ostendorf-cloud/ostendorf.tech/pull/153))
- **feed:** page through every Instagram post ([#148](https://github.com/ostendorf-cloud/ostendorf.tech/pull/148))
- **web:** serve the portrait at its original 1200 px quality ([#146](https://github.com/ostendorf-cloud/ostendorf.tech/pull/146))
- **web:** rework the Leistungen section as a calm card grid ([#142](https://github.com/ostendorf-cloud/ostendorf.tech/pull/142))
- **web:** glowing buttons and section backgrounds as on the old site ([#141](https://github.com/ostendorf-cloud/ostendorf.tech/pull/141))
- **web:** add a load more button to the Impressionen section ([#136](https://github.com/ostendorf-cloud/ostendorf.tech/pull/136))
- **web:** scroll the band carousel one card at a time ([#133](https://github.com/ostendorf-cloud/ostendorf.tech/pull/133))
- **compose:** run the real feed service under a realfeed profile ([#128](https://github.com/ostendorf-cloud/ostendorf.tech/pull/128))
- **web:** band order, one line headline and band pictures ([#125](https://github.com/ostendorf-cloud/ostendorf.tech/pull/125))
- **feed:** log in to OpenBao with kubernetes or a token, configurable secret keys ([#126](https://github.com/ostendorf-cloud/ostendorf.tech/pull/126))
- **web:** old site palette, Impressionen nav entry, tile hover, slow hero beams ([#121](https://github.com/ostendorf-cloud/ostendorf.tech/pull/121))
- **web:** reworked band texts, band colours and a self scrolling carousel ([#114](https://github.com/ostendorf-cloud/ostendorf.tech/pull/114))
- **compose:** add a fake Instagram feed for local testing ([#116](https://github.com/ostendorf-cloud/ostendorf.tech/pull/116))
- use govue's contact form, drop the session cookie and the CSRF key ([#115](https://github.com/ostendorf-cloud/ostendorf.tech/pull/115))
- **web:** compact mobile header, colour mode icons, hero fixes and bands carousel ([#108](https://github.com/ostendorf-cloud/ostendorf.tech/pull/108))
- **web:** add a favicon and a social preview image ([#96](https://github.com/ostendorf-cloud/ostendorf.tech/pull/96))
- **analytics:** count page views with govue instead of site code ([#88](https://github.com/ostendorf-cloud/ostendorf.tech/pull/88))
- **legal:** serve the legal pages with the govue routes ([#86](https://github.com/ostendorf-cloud/ostendorf.tech/pull/86))
- **web:** bring back the lit stage, the brand gradient and the icons ([#61](https://github.com/ostendorf-cloud/ostendorf.tech/pull/61))
- **chart:** optional feed workload and scoped egress ([#59](https://github.com/ostendorf-cloud/ostendorf.tech/pull/59))
- a feed service pod that owns the Instagram token ([#51](https://github.com/ostendorf-cloud/ostendorf.tech/pull/51))
- **chart:** add a Gateway API HTTPRoute alongside the Ingress ([#48](https://github.com/ostendorf-cloud/ostendorf.tech/pull/48))
- **webapp:** govue v0.4.0 and the operational endpoints on an admin listener ([#45](https://github.com/ostendorf-cloud/ostendorf.tech/pull/45))
- the Instagram feed and the Umami hookup ([#23](https://github.com/ostendorf-cloud/ostendorf.tech/pull/23))
- contact form, container, chart and publishing ([#6](https://github.com/ostendorf-cloud/ostendorf.tech/pull/6))
- **web:** circular theme reveal and counting facts ([#5](https://github.com/ostendorf-cloud/ostendorf.tech/pull/5))
- **web:** prerender every route in every locale
- **web:** add the home, legal and not found pages
- **web:** add the header, footer, theme toggle and locale switch
- **web:** add the logo mark and wordmark
- **web:** add the german and english message catalogues
- **web:** add the branded vtech theme with a contrast check
- **web:** scaffold the vue application on govue
- **website:** add the entry point and the embedded assets
- **website:** add the http handler and page serving
- **website:** add configuration loading

### Bug Fixes

- **deps:** update govue to v0.10.1 ([#157](https://github.com/ostendorf-cloud/ostendorf.tech/pull/157))
- **deps:** update module go.yaml.in/yaml/v3 to v4 ([#124](https://github.com/ostendorf-cloud/ostendorf.tech/pull/124))
- **renovate:** raise both govue halves in one pull request ([#152](https://github.com/ostendorf-cloud/ostendorf.tech/pull/152))
- **feedclient:** retry a failed feed sync within seconds ([#138](https://github.com/ostendorf-cloud/ostendorf.tech/pull/138))
- **build:** run web:build and prerender once per task run ([#139](https://github.com/ostendorf-cloud/ostendorf.tech/pull/139))
- **web:** thin the light mode hero scrim to show more photo ([#134](https://github.com/ostendorf-cloud/ostendorf.tech/pull/134))
- **web:** stop the gradient headline from clipping its letters ([#127](https://github.com/ostendorf-cloud/ostendorf.tech/pull/127))
- **web:** animate the hero logo again ([#132](https://github.com/ostendorf-cloud/ostendorf.tech/pull/132))
- **deps:** update module go.yaml.in/yaml/v3 to v3.0.6 ([#118](https://github.com/ostendorf-cloud/ostendorf.tech/pull/118))
- one port variable for compose, complete env docs, chart guard, SEO details ([#106](https://github.com/ostendorf-cloud/ostendorf.tech/pull/106))
- **pow:** end the salt with a separator and require it ([#94](https://github.com/ostendorf-cloud/ostendorf.tech/pull/94))
- **website:** make the embedded directory check actually check ([#71](https://github.com/ostendorf-cloud/ostendorf.tech/pull/71))
- **deps:** wait out pnpm's minimum release age before raising updates ([#64](https://github.com/ostendorf-cloud/ostendorf.tech/pull/64))
- **docker:** pin base images by manifest list digest ([#60](https://github.com/ostendorf-cloud/ostendorf.tech/pull/60))
- **chart:** make lint:helm actually check the render-time guards ([#57](https://github.com/ostendorf-cloud/ostendorf.tech/pull/57))
- **deps:** update dependency vue-router to v5 ([#43](https://github.com/ostendorf-cloud/ostendorf.tech/pull/43))
- close every deferred finding from the go-live review ([#22](https://github.com/ostendorf-cloud/ostendorf.tech/pull/22))
- **renovate:** pinDigests is a config key, not a preset ([#21](https://github.com/ostendorf-cloud/ostendorf.tech/pull/21))
- **routes:** reject a route table with duplicate names or files ([#13](https://github.com/ostendorf-cloud/ostendorf.tech/pull/13))
- make the single route table real rather than aspirational ([#9](https://github.com/ostendorf-cloud/ostendorf.tech/pull/9))
- **web:** correct the imprint and review the legal pages ([#4](https://github.com/ostendorf-cloud/ostendorf.tech/pull/4))
