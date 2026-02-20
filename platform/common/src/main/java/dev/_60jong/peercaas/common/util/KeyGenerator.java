package dev._60jong.peercaas.common.util;

import java.util.UUID;

public class KeyGenerator {

    public static String generate() {
        return UUID.randomUUID().toString();
    }
}
