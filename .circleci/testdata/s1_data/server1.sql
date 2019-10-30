-- MySQL dump 10.13  Distrib 5.7.26, for Linux (x86_64)
--
-- Host: localhost    Database: jackal
-- ------------------------------------------------------
-- Server version	5.7.26-0ubuntu0.16.04.1

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `blocklist_items`
--

DROP TABLE IF EXISTS `blocklist_items`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `blocklist_items` (
  `username` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL,
  `jid` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`username`,`jid`),
  KEY `i_blocklist_items_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `blocklist_items`
--

LOCK TABLES `blocklist_items` WRITE;
/*!40000 ALTER TABLE `blocklist_items` DISABLE KEYS */;
/*!40000 ALTER TABLE `blocklist_items` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `offline_messages`
--

DROP TABLE IF EXISTS `offline_messages`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `offline_messages` (
  `username` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL,
  `data` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime NOT NULL,
  KEY `i_offline_messages_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `offline_messages`
--

LOCK TABLES `offline_messages` WRITE;
/*!40000 ALTER TABLE `offline_messages` DISABLE KEYS */;
/*!40000 ALTER TABLE `offline_messages` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `private_storage`
--

DROP TABLE IF EXISTS `private_storage`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `private_storage` (
  `username` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL,
  `namespace` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL,
  `data` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `updated_at` datetime NOT NULL,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`username`,`namespace`),
  KEY `i_private_storage_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `private_storage`
--

LOCK TABLES `private_storage` WRITE;
/*!40000 ALTER TABLE `private_storage` DISABLE KEYS */;
/*!40000 ALTER TABLE `private_storage` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `roster_groups`
--

DROP TABLE IF EXISTS `roster_groups`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `roster_groups` (
  `username` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL,
  `jid` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL,
  `group` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `updated_at` datetime NOT NULL,
  `created_at` datetime NOT NULL,
  KEY `i_roster_groups_username_jid` (`username`,`jid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `roster_groups`
--

LOCK TABLES `roster_groups` WRITE;
/*!40000 ALTER TABLE `roster_groups` DISABLE KEYS */;
/*!40000 ALTER TABLE `roster_groups` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `roster_items`
--

DROP TABLE IF EXISTS `roster_items`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `roster_items` (
  `username` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL,
  `jid` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `subscription` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `groups` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `ask` tinyint(1) NOT NULL,
  `ver` int(11) NOT NULL DEFAULT '0',
  `updated_at` datetime NOT NULL,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`username`,`jid`),
  KEY `i_roster_items_username` (`username`),
  KEY `i_roster_items_jid` (`jid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `roster_items`
--

LOCK TABLES `roster_items` WRITE;
/*!40000 ALTER TABLE `roster_items` DISABLE KEYS */;
/*!40000 ALTER TABLE `roster_items` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `roster_notifications`
--

DROP TABLE IF EXISTS `roster_notifications`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `roster_notifications` (
  `contact` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL,
  `jid` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL,
  `elements` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `updated_at` datetime NOT NULL,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`contact`,`jid`),
  KEY `i_roster_notifications_jid` (`jid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `roster_notifications`
--

LOCK TABLES `roster_notifications` WRITE;
/*!40000 ALTER TABLE `roster_notifications` DISABLE KEYS */;
/*!40000 ALTER TABLE `roster_notifications` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `roster_versions`
--

DROP TABLE IF EXISTS `roster_versions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `roster_versions` (
  `username` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL,
  `ver` int(11) NOT NULL DEFAULT '0',
  `last_deletion_ver` int(11) NOT NULL DEFAULT '0',
  `updated_at` datetime NOT NULL,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `roster_versions`
--

LOCK TABLES `roster_versions` WRITE;
/*!40000 ALTER TABLE `roster_versions` DISABLE KEYS */;
/*!40000 ALTER TABLE `roster_versions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `users`
--

DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `users` (
  `username` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL,
  `password` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `last_presence` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `last_presence_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `users`
--

LOCK TABLES `users` WRITE;
/*!40000 ALTER TABLE `users` DISABLE KEYS */;
INSERT INTO `users` VALUES ('user1','asdf','<presence from=\"user1@server1.xmpp/profanity\" to=\"user1@server1.xmpp\" type=\"unavailable\"/>','2019-06-20 00:28:08','2019-06-20 00:28:08','2019-04-19 18:42:58');
/*!40000 ALTER TABLE `users` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `vcards`
--

DROP TABLE IF EXISTS `vcards`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `vcards` (
  `username` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL,
  `vcard` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `updated_at` datetime NOT NULL,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `vcards`
--

LOCK TABLES `vcards` WRITE;
/*!40000 ALTER TABLE `vcards` DISABLE KEYS */;
/*!40000 ALTER TABLE `vcards` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2019-07-03  1:13:58
