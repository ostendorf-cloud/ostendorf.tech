# Changelog

## v0.1.0 (2026-11-21)

### Features

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
