# SPDX-FileCopyrightText: 2020 Ethel Morgan
#
# SPDX-License-Identifier: MIT

{ pkgs ? import <nixpkgs> {} }:
with pkgs;

stdenv.mkDerivation rec {
  name = "catbus-web-ui-${version}";
  version = "latest"; 

  src = lib.sourceFilesBySuffices ./. [
    ".html"
    ".js"
    ".png"
    ".svg"
  ];

  installPhase = ''
    mkdir -p $out
    cp -r ${src}/* $out
  '';
}
