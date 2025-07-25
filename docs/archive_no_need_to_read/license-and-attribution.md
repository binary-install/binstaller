---
title: "License and Attribution"
date: "2025-04-17"
author: "haya14busa"
version: "0.1.0"
status: "draft"
---

# License and Attribution

This document outlines the license information for the GoDownloader fork and provides attribution to the original project and its contributors.

## License

The GoDownloader fork is licensed under the MIT License, the same license as the original project. This permissive license allows for free use, modification, and distribution of the software, subject to the conditions outlined below.

### MIT License

```
MIT License

Copyright (c) 2025 haya14busa

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

## Original Project Attribution

This project is a fork of [GoDownloader](https://github.com/goreleaser/godownloader), which was created and maintained by the GoReleaser team. We acknowledge and appreciate their work in creating the original project.

### Original Project Information

- **Project Name**: GoDownloader
- **Repository**: https://github.com/goreleaser/godownloader
- **License**: MIT License
- **Copyright**: Copyright (c) 2017-2022 The GoReleaser Authors

### Original Project Contributors

We would like to acknowledge the contributions of the original GoDownloader authors and contributors, including but not limited to:

- Carlos Alexandro Becker ([@caarlos0](https://github.com/caarlos0))
- Nick Galbreath ([@client9](https://github.com/client9))
- Matt Farina ([@mattfarina](https://github.com/mattfarina))
- And many others who contributed to the original project

Their work laid the foundation for this fork, and we are grateful for their efforts.

## Third-Party Dependencies

The GoDownloader fork uses several third-party dependencies, each with its own license. The major dependencies and their licenses are listed below:

| Dependency | License | Repository |
|------------|---------|------------|
| goreleaser/goreleaser | MIT | https://github.com/goreleaser/goreleaser |
| apex/log | MIT | https://github.com/apex/log |
| client9/codegen | MIT | https://github.com/client9/codegen |
| pkg/errors | BSD-2-Clause | https://github.com/pkg/errors |
| alecthomas/kingpin.v2 | MIT | https://github.com/alecthomas/kingpin |
| yaml.v2 | Apache-2.0 | https://github.com/go-yaml/yaml |

For a complete list of dependencies and their licenses, please refer to the `go.mod` file and the respective repositories.

## Shell Script Attribution

The shell scripts generated by GoDownloader include functions from [shlib](https://github.com/client9/shlib), which is in the public domain under the [Unlicense](http://unlicense.org/). We acknowledge and appreciate this contribution.

```
https://github.com/client9/shlib - portable posix shell functions
Public domain - http://unlicense.org
https://github.com/client9/shlib/blob/master/LICENSE.md
but credit (and pull requests) appreciated.
```

## Modifications and Additions

This fork includes several modifications and additions to the original project:

1. **Removed Features**:
   - Equinox.io support
   - Raw GitHub releases support
   - Tree walking functionality

2. **Added Features**:
   - GitHub attestation verification
   - Enhanced documentation
   - Improved error handling
   - Updated dependencies

These modifications and additions are also licensed under the MIT License.

## Contribution Attribution

We are grateful to all contributors to this fork. Contributors are listed in the [CONTRIBUTORS.md](../CONTRIBUTORS.md) file and acknowledged in release notes.

## Using and Attributing This Project

If you use this project in your own work, we request that you provide attribution by:

1. Retaining the copyright and license notices in the source code
2. Mentioning the project name and repository URL in your documentation
3. Acknowledging the original GoDownloader project and its contributors

Example attribution in documentation:

```markdown
This project uses [GoDownloader Fork](https://github.com/haya14busa/godownloader),
a fork of [GoDownloader](https://github.com/goreleaser/godownloader),
licensed under the MIT License.
```

## Contact Information

If you have any questions about licensing or attribution, please contact the project maintainer:

- GitHub: [@haya14busa](https://github.com/haya14busa)

## License Compliance

We strive to ensure that this project complies with all license requirements of its dependencies and the original project. If you identify any compliance issues, please report them by opening an issue on the project repository.

## Disclaimer

This document is provided for informational purposes only and does not constitute legal advice. For specific legal questions regarding licensing and attribution, please consult a legal professional.
