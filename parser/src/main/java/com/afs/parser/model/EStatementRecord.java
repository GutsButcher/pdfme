package com.afs.parser.model;

import java.text.DecimalFormat;
import java.time.LocalDate;
import java.time.format.DateTimeFormatter;
import java.util.ArrayList;
import java.util.List;

public class EStatementRecord {

    private String orgId;
    private String cardNumber;
    private String statementDate;
    private String name;
    private String address;
    private double availableBalance;
    private double openingBalance;
    private double toatalDepits;
    private double totalCredits;
    private double currentBalance;
    private List<Transaction> transactions;

    public EStatementRecord() {
        this.transactions = new ArrayList<>();
    }


    public String getOrgId() {
        return orgId;
    }

    public void setOrgId(String orgId) {
        this.orgId = orgId;
    }

    public String getCardNumber() {
        return cardNumber;
    }

    public void setCardNumber(String cardNumber) {
        this.cardNumber = cardNumber;
    }

    public String getStatementDate() {
        return statementDate;
    }

    public void setStatementDate(String statementDate) {
        this.statementDate = statementDate;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public String getAddress() {
        return address;
    }

    public void setAddress(String address) {
        this.address = address;
    }

    public double getAvailableBalance() {
        return availableBalance;
    }

    public void setAvailableBalance(double availableBalance) {
        this.availableBalance = availableBalance;
    }

    public double getOpeningBalance() {
        return openingBalance;
    }

    public void setOpeningBalance(double openingBalance) {
        this.openingBalance = openingBalance;
    }

    public double getToatalDepits() {
        return toatalDepits;
    }

    public void setToatalDepits(double toatalDepits) {
        this.toatalDepits = toatalDepits;
    }

    public double getTotalCredits() {
        return totalCredits;
    }

    public void setTotalCredits(double totalCredits) {
        this.totalCredits = totalCredits;
    }

    public double getCurrentBalance() {
        return currentBalance;
    }

    public void setCurrentBalance(double currentBalance) {
        this.currentBalance = currentBalance;
    }

    public List<Transaction> getTransactions() {
        return transactions;
    }

    public void setTransactions(List<Transaction> transactions) {
        this.transactions = transactions;
    }

    public void addTransaction(Transaction transaction) {
        this.transactions.add(transaction);
    }

    public static String formatDate(String inputDate) {
        if(inputDate.equals("00000000")){
            return null;
        }
        DateTimeFormatter inputFormatter = DateTimeFormatter.ofPattern("ddMMyyyy");
        DateTimeFormatter outputFormatter = DateTimeFormatter.ofPattern("dd/MM/yyyy");
        LocalDate date = LocalDate.parse(inputDate, inputFormatter);
        return date.format(outputFormatter);
    }

    public static double ParseDouble(String rawValue) {

        if (rawValue == null || rawValue.trim().isEmpty()) {
            throw new NumberFormatException("Input string is null or empty");
        }

        String cleaned = rawValue.trim();

        if (cleaned.endsWith("-")) {
            cleaned = "-" + cleaned.substring(0, cleaned.length() - 1);
        }

        cleaned = cleaned.replaceAll("[^0-9.\\-]", "");

        return Double.parseDouble(cleaned);
    }

    public static double ParseAmount(String rawValue){
        double amount = ParseDouble(rawValue);
        double result = (amount % 1 == 0) ? amount / 1000 : amount;
        DecimalFormat df = new DecimalFormat("0.000");
        return Double.parseDouble(df.format(result));
    }


    @Override
    public String toString() {
        return "EstatmentRecord{" +
                "orgId='" + orgId + '\'' +
                ", cardNumber='" + cardNumber + '\'' +
                ", statementDate='" + statementDate + '\'' +
                ", name='" + name + '\'' +
                ", address='" + address + '\'' +
                ", availableBalance=" + availableBalance +
                ", openingBalance=" + openingBalance +
                ", toatalDepits=" + toatalDepits +
                ", totalCredits=" + totalCredits +
                ", currentBalance=" + currentBalance +
                '}';
    }

}