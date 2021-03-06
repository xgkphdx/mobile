// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

// TODO(crawshaw): TestBindIOS

func TestImportPackagesPathCleaning(t *testing.T) {
	slashPath := "golang.org/x/mobile/example/bind/hello/"
	pkgs, err := importPackages([]string{slashPath})
	if err != nil {
		t.Fatal(err)
	}
	p := pkgs[0]
	if c := path.Clean(slashPath); p.ImportPath != c {
		t.Errorf("expected %s; got %s", c, p.ImportPath)
	}
}

func TestBindAndroid(t *testing.T) {
	androidHome := os.Getenv("ANDROID_HOME")
	if androidHome == "" {
		t.Skip("ANDROID_HOME not found, skipping bind")
	}
	platform, err := androidAPIPath()
	if err != nil {
		t.Skip("No android API platform found in $ANDROID_HOME, skipping bind")
	}
	platform = strings.Replace(platform, androidHome, "$ANDROID_HOME", -1)

	defer func() {
		xout = os.Stderr
		buildN = false
		buildX = false
		buildO = ""
		buildTarget = ""
	}()
	buildN = true
	buildX = true
	buildO = "asset.aar"
	buildTarget = "android/arm"
	ndkRoot = "/NDK"

	tests := []struct {
		javaPkg    string
		wantGobind string
		wantPkgDir string
	}{
		{
			wantGobind: "gobind -lang=java",
			wantPkgDir: "asset",
		},
		{
			javaPkg:    "com.example.foo",
			wantGobind: "gobind -lang=java -javapkg=com.example.foo",
			wantPkgDir: "com/example/foo/asset",
		},
	}
	for _, tc := range tests {
		bindJavaPkg = tc.javaPkg

		buf := new(bytes.Buffer)
		xout = buf
		gopath = filepath.SplitList(os.Getenv("GOPATH"))[0]
		if goos == "windows" {
			os.Setenv("HOMEDRIVE", "C:")
		}
		cmdBind.flag.Parse([]string{"golang.org/x/mobile/asset"})
		err := runBind(cmdBind)
		if err != nil {
			t.Log(buf.String())
			t.Fatal(err)
		}
		got := filepath.ToSlash(buf.String())

		data := struct {
			outputData
			AndroidPlatform string
			GobindJavaCmd   string
			JavaPkgDir      string
		}{
			outputData:      defaultOutputData(),
			AndroidPlatform: platform,
			GobindJavaCmd:   tc.wantGobind,
			JavaPkgDir:      tc.wantPkgDir,
		}

		wantBuf := new(bytes.Buffer)
		if err := bindAndroidTmpl.Execute(wantBuf, data); err != nil {
			t.Errorf("%+v: computing diff failed: %v", tc, err)
			continue
		}

		diff, err := diff(got, wantBuf.String())
		if err != nil {
			t.Errorf("%+v: computing diff failed: %v", tc, err)
			continue
		}
		if diff != "" {
			t.Errorf("%+v: unexpected output:\n%s", tc, diff)
		}
	}
}

var bindAndroidTmpl = template.Must(template.New("output").Parse(`GOMOBILE={{.GOPATH}}/pkg/gomobile
WORK=$WORK
mkdir -p $WORK/gomobile_bind
mkdir -p $WORK/gomobile_bind
mkdir -p $WORK/gomobile_bind
mkdir -p $WORK/gomobile_bind
mkdir -p $WORK/gen/src/Java
GOOS=android GOARCH=arm CC=/NDK/toolchains/llvm/prebuilt/{{.GOOS}}-{{.NDKARCH}}/bin/clang{{.EXE}} CXX=/NDK/toolchains/llvm/prebuilt/{{.GOOS}}-{{.NDKARCH}}/bin/clang++{{.EXE}} CGO_CFLAGS=-target armv7a-none-linux-androideabi --sysroot /NDK/platforms/android-15/arch-arm -gcc-toolchain /NDK/toolchains/arm-linux-androideabi-4.9/prebuilt/{{.GOOS}}-{{.NDKARCH}} -I$GOMOBILE/include CGO_CPPFLAGS=-target armv7a-none-linux-androideabi --sysroot /NDK/platforms/android-15/arch-arm -gcc-toolchain /NDK/toolchains/arm-linux-androideabi-4.9/prebuilt/{{.GOOS}}-{{.NDKARCH}} -I$GOMOBILE/include CGO_LDFLAGS=-target armv7a-none-linux-androideabi --sysroot /NDK/platforms/android-15/arch-arm -gcc-toolchain /NDK/toolchains/arm-linux-androideabi-4.9/prebuilt/{{.GOOS}}-{{.NDKARCH}} -L/NDK/platforms/android-15/arch-arm/usr/lib -L$GOMOBILE/lib/arm CGO_ENABLED=1 GOARM=7 GOPATH=$WORK/gen:$GOPATH go install -pkgdir=$GOMOBILE/pkg_android_arm -x -gcflags=-shared -ldflags=-shared golang.org/x/mobile/asset
rm -r -f "$WORK/fakegopath"
mkdir -p $WORK/fakegopath/pkg
cp $GOMOBILE/pkg_android_arm/golang.org/x/mobile/asset.a $WORK/fakegopath/pkg/android_arm/golang.org/x/mobile/asset.a
mkdir -p $WORK/fakegopath/pkg/android_arm/golang.org/x/mobile
mkdir -p $WORK/gomobile_bind
gobind -lang=go -outdir=$WORK/gomobile_bind golang.org/x/mobile/asset
mkdir -p $WORK/gomobile_bind
gobind -lang=go -outdir=$WORK/gomobile_bind 
mkdir -p $WORK/androidlib
mkdir -p $WORK/android/src/main/java/{{.JavaPkgDir}}
{{.GobindJavaCmd}} -outdir=$WORK/android/src/main/java/{{.JavaPkgDir}} golang.org/x/mobile/asset
mkdir -p $WORK/gomobile_bind
mkdir -p $WORK/gomobile_bind
mkdir -p $WORK/android/src/main/java/go
{{.GobindJavaCmd}} -outdir=$WORK/android/src/main/java/go 
mkdir -p $WORK/android/src/main/java/go
mkdir -p $WORK/gomobile_bind
mkdir -p $WORK/gomobile_bind
cp $GOPATH/src/golang.org/x/mobile/bind/java/seq_android.go.support $WORK/gomobile_bind/seq_android.go
mkdir -p $WORK/gomobile_bind
cp $GOPATH/src/golang.org/x/mobile/bind/java/seq_android.c.support $WORK/gomobile_bind/seq_android.c
mkdir -p $WORK/gomobile_bind
cp $GOPATH/src/golang.org/x/mobile/bind/java/seq.h $WORK/gomobile_bind/seq.h
mkdir -p $WORK/gomobile_bind
cp $GOPATH/src/golang.org/x/mobile/bind/seq.go.support $WORK/gomobile_bind/seq.go
mkdir -p $WORK/gomobile_bind
mkdir -p $WORK/android/src/main/java/go
GOOS=android GOARCH=arm CC=/NDK/toolchains/llvm/prebuilt/{{.GOOS}}-{{.NDKARCH}}/bin/clang{{.EXE}} CXX=/NDK/toolchains/llvm/prebuilt/{{.GOOS}}-{{.NDKARCH}}/bin/clang++{{.EXE}} CGO_CFLAGS=-target armv7a-none-linux-androideabi --sysroot /NDK/platforms/android-15/arch-arm -gcc-toolchain /NDK/toolchains/arm-linux-androideabi-4.9/prebuilt/{{.GOOS}}-{{.NDKARCH}} -I$GOMOBILE/include CGO_CPPFLAGS=-target armv7a-none-linux-androideabi --sysroot /NDK/platforms/android-15/arch-arm -gcc-toolchain /NDK/toolchains/arm-linux-androideabi-4.9/prebuilt/{{.GOOS}}-{{.NDKARCH}} -I$GOMOBILE/include CGO_LDFLAGS=-target armv7a-none-linux-androideabi --sysroot /NDK/platforms/android-15/arch-arm -gcc-toolchain /NDK/toolchains/arm-linux-androideabi-4.9/prebuilt/{{.GOOS}}-{{.NDKARCH}} -L/NDK/platforms/android-15/arch-arm/usr/lib -L$GOMOBILE/lib/arm CGO_ENABLED=1 GOARM=7 GOPATH=$WORK/gen:$GOPATH go build -pkgdir=$GOMOBILE/pkg_android_arm -x -buildmode=c-shared -o=$WORK/android/src/main/jniLibs/armeabi-v7a/libgojni.so $WORK/androidlib/main.go
rm $WORK/android/src/main/java/go/Seq.java
ln -s $GOPATH/src/golang.org/x/mobile/bind/java/Seq.java $WORK/android/src/main/java/go/Seq.java
rm $WORK/android/src/main/java/go/LoadJNI.java
ln -s $GOPATH/src/golang.org/x/mobile/bind/java/LoadJNI.java $WORK/android/src/main/java/go/LoadJNI.java
PWD=$WORK/android/src/main/java javac -d $WORK/javac-output -source 1.7 -target 1.7 -bootclasspath {{.AndroidPlatform}}/android.jar *.java
jar c -C $WORK/javac-output .
`))
