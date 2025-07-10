package checker

import (
    "testing"
)

func TestFunctionBlock(t *testing.T) {
    fb := &FunctionBlock{}

    // Test Base64 Encode
    encoded, err := fb.Apply(FuncBase64Encode, "Hello, World!")
    if err != nil || encoded != "SGVsbG8sIFdvcmxkIQ==" {
        t.Errorf("Base64Encode failed. Expected: SGVsbG8sIFdvcmxkIQ==, Got: %s, Err: %v", encoded, err)
    }

    // Test Base64 Decode
    decoded, err := fb.Apply(FuncBase64Decode, "SGVsbG8sIFdvcmxkIQ==")
    if err != nil || decoded != "Hello, World!" {
        t.Errorf("Base64Decode failed. Expected: Hello, World!, Got: %s, Err: %v", decoded, err)
    }

    // Test SHA256
    sha256Result, err := fb.Apply(FuncSHA256, "test")
    expectedSHA256 := "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
    if err != nil || sha256Result != expectedSHA256 {
        t.Errorf("SHA256 failed. Expected: %s, Got: %s, Err: %v", expectedSHA256, sha256Result, err)
    }

    // Test MD5
    md5Result, err := fb.Apply(FuncMD5, "test")
    expectedMD5 := "098f6bcd4621d373cade4e832627b4f6"
    if err != nil || md5Result != expectedMD5 {
        t.Errorf("MD5 failed. Expected: %s, Got: %s, Err: %v", expectedMD5, md5Result, err)
    }

    // Test URLEncode
    urlEncoded, err := fb.Apply(FuncURLEncode, "a b+c")
    expectedURLEncode := "a+b%2Bc"
    if err != nil || urlEncoded != expectedURLEncode {
        t.Errorf("URLEncode failed. Expected: %s, Got: %s, Err: %v", expectedURLEncode, urlEncoded, err)
    }

    // Test URLDecode
    urlDecoded, err := fb.Apply(FuncURLDecode, "a+b%2Bc")
    if err != nil || urlDecoded != "a b+c" {
        t.Errorf("URLDecode failed. Expected: a b+c, Got: %s, Err: %v", urlDecoded, err)
    }

    // Test ToUpper
    upper, err := fb.Apply(FuncToUpper, "hello")
    expectedUpper := "HELLO"
    if err != nil || upper != expectedUpper {
        t.Errorf("ToUpper failed. Expected: %s, Got: %s, Err: %v", expectedUpper, upper, err)
    }

    // Test ToLower
    lower, err := fb.Apply(FuncToLower, "HELLO")
    expectedLower := "hello"
    if err != nil || lower != expectedLower {
        t.Errorf("ToLower failed. Expected: %s, Got: %s, Err: %v", expectedLower, lower, err)
    }
}

func TestVariableManipulator(t *testing.T) {
    vars := NewVariableList()
    vm := NewVariableManipulator(vars)

    // Test setting and replacing a variable
    vm.SetVariable("greeting", "Hello, World!", false)
    replaced := vm.ReplaceVariables("Say: <greeting>")
    expected := "Say: Hello, World!"
    if replaced != expected {
        t.Errorf("ReplaceVariables failed. Expected: %s, Got: %s", expected, replaced)
    }

    // Test TransformVariable (to upper)
    err := vm.TransformVariable("greeting", FuncToUpper)
    transformedVar, _ := vm.variables.Get("greeting")

    if err != nil || transformedVar.Value != "HELLO, WORLD!" {
        t.Errorf("TransformVariable failed. Expected: HELLO, WORLD!, Got: %v, Err: %v", transformedVar.Value, err)
    }

    // Test TransformVariable on invalid variable
    err = vm.TransformVariable("nonexistent", FuncToUpper)
    if err == nil {
        t.Error("TransformVariable should have failed for nonexistent variable")
    }
}
