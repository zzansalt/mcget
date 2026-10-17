# mcget

`mcget` is a terminal application for selecting and downloading Minecraft server software. It supports Paper, Velocity, Spigot, Purpur, Forge, NeoForge, and Fabric, with English and Korean menus.

## Requirements

- Go 1.25 or later to build from source.
- An internet connection to retrieve server versions, builds, and files.
- Java available on `PATH` when installing Forge, NeoForge, or Fabric servers.
- For the folder picker: PowerShell on Windows, AppleScript (`osascript`) on macOS, or `zenity`/`kdialog` on Linux. You can enter a folder path manually if a Linux picker is unavailable.

## Run from source

```sh
git clone https://github.com/zzansalt/mcget.git
cd mcget
go run .
```

Build a native executable:

```sh
go build -o mcget .
```

On Windows, use `go build -o mcget.exe .`.

## Use

1. Choose **Download Server**.
2. Select the server software, Minecraft version, and build or loader version.
3. Select an installation folder or enter its path manually.
4. For an existing download filename, choose whether to overwrite it, keep the existing file, or save a numbered copy.

Press `q` to cancel an active download or return to the previous menu. The language can be changed from **Language**; English is the default. Locale strings are stored in `models/locales/en.json` and `models/locales/ko.json`.

Forge, NeoForge, and Fabric use their installers to create server files in the selected folder. Forge and NeoForge installers may create platform run scripts instead of a standalone server JAR. SHA-256 is checked when the provider publishes a checksum.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE).

---

# mcget (한국어)

`mcget`은 Minecraft 서버 소프트웨어를 선택하고 다운로드하는 터미널 앱입니다. Paper, Velocity, Spigot, Purpur, Forge, NeoForge, Fabric을 지원하며 영어와 한국어 메뉴를 제공합니다.

## 요구 사항

- 소스에서 빌드하려면 Go 1.25 이상이 필요합니다.
- 서버 버전과 파일 정보를 가져오려면 인터넷 연결이 필요합니다.
- Forge, NeoForge, Fabric 서버를 설치하려면 `PATH`에서 Java를 찾을 수 있어야 합니다.
- 폴더 선택 대화상자는 Windows에서 PowerShell, macOS에서 `osascript`, Linux에서 `zenity` 또는 `kdialog`를 사용합니다. Linux에서 대화상자를 사용할 수 없으면 경로를 직접 입력할 수 있습니다.

## 소스에서 실행

```sh
git clone https://github.com/zzansalt/mcget.git
cd mcget
go run .
```

현재 운영체제용 실행 파일을 빌드하려면 다음을 실행합니다.

```sh
go build -o mcget .
```

Windows에서는 `go build -o mcget.exe .`를 사용합니다.

## 사용 방법

1. **Download Server**를 선택합니다.
2. 서버 소프트웨어, Minecraft 버전, 빌드 또는 로더 버전을 선택합니다.
3. 설치 폴더를 선택하거나 경로를 직접 입력합니다.
4. 같은 이름의 파일이 있으면 덮어쓰기, 기존 파일 유지, 번호를 붙여 별도 저장 중 하나를 선택합니다.

다운로드 중 `q`를 누르면 취소할 수 있습니다. **Language** 메뉴에서 언어를 바꿀 수 있으며 기본 언어는 영어입니다. 번역 문구는 `models/locales/en.json`과 `models/locales/ko.json`에 있습니다.

Forge, NeoForge, Fabric은 각 설치 프로그램으로 선택한 폴더에 서버 파일을 설치합니다. Forge와 NeoForge는 단독 서버 JAR 대신 운영체제별 실행 스크립트를 만들 수 있습니다. 제공자가 체크섬을 공개한 경우 SHA-256을 검증합니다.

## 라이선스

MIT License로 배포됩니다. 자세한 내용은 [LICENSE](LICENSE)를 확인하세요.
