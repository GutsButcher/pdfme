package com.afs.parser.service;

import com.afs.parser.model.EStatementRecord;
import com.afs.parser.model.Transaction;

import java.io.*;
import java.nio.file.*;

public class StatementParser {

    public static EStatementRecord StatementParser(String filePath) throws IOException {

        String content = Files.readString(Paths.get(filePath));
        EStatementRecord record = new  EStatementRecord();


//      This will seperate eachrow on its own
        String[] rawRecords = content.split("\\r?\\n");

//      this will go over each line
        outerLoop:
        for (String recordLine : rawRecords) {
//          this will seperate each field by | dillemeter
            String[] fields = recordLine.split("\\|");

            String recordType = fields[3].trim();
          // will go over the record types to filter the parsing data based on it
            switch (recordType) {
                // Account Information(Limit, OTB, due, dates etc)
                case "1":

                    // orgid
                    record.setOrgId(fields[0].trim());

                    // statment Date
                    String date = EStatementRecord.formatDate(fields[14].trim());
                    record.setStatementDate(date);

                    // card number with removed first two 00-
                    record.setCardNumber(fields[2].trim().substring(3));

                    // current balance
                    double cBalance = EStatementRecord.ParseDouble(fields[12].trim());
                    record.setCurrentBalance(cBalance);

                    // opening balance
                    double oBalance = EStatementRecord.ParseDouble(fields[45].trim());
                    record.setOpeningBalance(oBalance);


                    // cridited amount
                    double cAmount = EStatementRecord.ParseDouble(fields[41].trim());
                    record.setTotalCredits(cAmount);

                    // depits amount
                    double dAmount = EStatementRecord.ParseDouble(fields[43].trim());
                    record.setToatalDepits(dAmount);

                    // available amount
                    double aAmount = EStatementRecord.ParseAmount(fields[27].trim());
                    record .setAvailableBalance(aAmount);

                    break;


                // Customer Information (Name address etc)
                case "2":

                    // name
                    record.setName(fields[5].trim());
                    // addreess
                    record.setAddress( fields[6].trim() + " " +fields[7].trim()+" "+ fields[8].trim()+" "+ fields[82].trim());
                    break;

                // This will skip Record Type 3,5
                case "3", "5":
                    break;

                // Transactions (date, merchant, amt etc)
                case "4":

                    if(fields[22].trim().equals("NEWL")){
                        break;
                    }


                    Transaction transaction = new Transaction();
                    transaction.setDate(EStatementRecord.formatDate(fields[4].trim()));
                    transaction.setPostDate(EStatementRecord.formatDate(fields[10].trim()));
                    transaction.setDescription(fields[22].trim());
                    transaction.setAmount(EStatementRecord.ParseAmount(fields[57].trim()));
                    transaction.setAmountInBHD(EStatementRecord.ParseAmount(fields[7].trim()));
                    transaction.setCurrency(fields[56].trim());

                    if(transaction.getDescription().equals("Payment Received")){
                        transaction.setCR(true);
                    }
                    record.addTransaction(transaction);
                    break;

                // Statement Messages (End of Customer data)
                case "6":
                    break outerLoop;
            }
        }
        return record;
    }

}
